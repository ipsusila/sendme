package sendme

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdmail "net/mail"
	"os"
	"sort"
	"strings"

	mail "github.com/xhit/go-simple-mail/v2"
)

// Html/Text message format
const (
	HtmlFormat  = "HTML"
	PlainFormat = "PLAIN"
)

// Action to be taken before sending.
// Options send email(Y/N/Abort)?
const (
	ActSend = iota
	ActSendAll
	ActDontSend
	ActAbortSend
	ActContinueError
)

// Executer for different template
type Executer interface {
	Execute(wr io.Writer, data any) error
}

// Mailer data structure
type Mailer struct {
	conf     *Config
	data     *MailDataCollection
	server   *mail.SMTPServer
	ccList   []string
	bccList  []string
	tpl      Executer
	ui       Ui
	sentWr   io.Writer
	sentList []string
}

func NewMailer(conf *Config) (*Mailer, error) {
	if conf == nil || conf.Server == nil || conf.Delivery == nil {
		return nil, errors.New("invalid/empty mail configuration")
	}

	// construct mailer
	m := Mailer{
		conf: conf,
	}
	var err error
	m.ui, err = NewUi(conf)
	if err != nil {
		return nil, err
	}

	// 1. Load data
	m.data, err = NewMailDataCollection(conf)
	if err != nil {
		return nil, err
	}

	// 2. Configure server
	m.server = mail.NewSMTPClient()
	if err := conf.Server.Configure(m.server); err != nil {
		return nil, err
	}
	m.server.TLSConfig, err = conf.Tls.MakeTlsConfig()
	if err != nil {
		return nil, err
	}

	// 3. Get templates
	switch conf.Delivery.MailFormat {
	case HtmlFormat:
		m.tpl, err = ParseHtmlTemplates(conf)
	case PlainFormat:
		m.tpl, err = ParseTextTemplates(conf)
	default:
		err = fmt.Errorf("unknown mail format: %s", conf.Delivery.MailFormat)
	}
	if err != nil {
		return nil, err
	}

	// parse cc/bcc
	if conf.Delivery.CcList != "" {
		cc, err := stdmail.ParseAddressList(conf.Delivery.CcList)
		if err != nil {
			return nil, err
		}
		for _, addr := range cc {
			m.ccList = append(m.ccList, addr.String())
		}
	}

	if conf.Delivery.BccList != "" {
		bcc, err := stdmail.ParseAddressList(conf.Delivery.BccList)
		if err != nil {
			return nil, err
		}
		for _, addr := range bcc {
			m.bccList = append(m.bccList, addr.String())
		}
	}

	if err := m.readSentList(); err != nil {
		return nil, err
	}

	return &m, nil
}

func (m *Mailer) readSentList() error {
	fd, err := os.Open(m.conf.Delivery.SentFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
	}
	defer fd.Close()

	scan := bufio.NewScanner(fd)
	for scan.Scan() {
		addr := strings.TrimSpace(scan.Text())
		if addr != "" {
			m.sentList = append(m.sentList, strings.ToLower(addr))
		}
	}
	sort.Strings(m.sentList)

	// log
	m.ui.Logf("SENT>>\n")
	for _, addr := range m.sentList {
		m.ui.Logf("  %s\n", addr)
	}

	return nil
}

func (m *Mailer) mailSent(addr string) bool {
	addr = strings.TrimSpace(addr)
	_, found := sort.Find(len(m.sentList), func(i int) int {
		return strings.Compare(addr, m.sentList[i])
	})

	return found
}

func (m *Mailer) Send(ctx context.Context) error {
	// ensure connection keep alive
	m.server.KeepAlive = true

	// open connection
	conn, err := m.server.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	// load send list
	fd, err := os.OpenFile(m.conf.Delivery.SentFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()
	m.sentWr = fd

	// loop through message and send email
	for _, datum := range m.data.Data {
		if !datum.HasFields(m.conf.Delivery.RequiredFields) {
			js, _ := json.Marshal(datum)
			m.ui.Logf("Skip DATUM>> %s\n", string(js))
			continue
		}
		var sb strings.Builder
		if err := m.tpl.Execute(&sb, datum); err != nil {
			js, _ := json.Marshal(datum)
			m.ui.Logf("DATUM>> %s\n", string(js))
			return err
		}

		// send each mail
		action, err := m.sendMail(ctx, conn, datum, sb.String())
		if err != nil && action != ActContinueError {
			return err
		}

		// Check action/message
		switch action {
		case ActAbortSend:
			return errors.New("aborted by user")
		case ActContinueError:
			if err != nil {
				m.ui.Logf("Error when sending email: %v\n", err)
			}
		case ActDontSend:
			// TODO:
		case ActSend:
			// TODO:
		}
	}

	return nil
}

func (m *Mailer) sendMail(ctx context.Context, conn *mail.SMTPClient, datum MailData, body string) (int, error) {
	c := m.conf
	msg := mail.NewMSG()

	// get to addr
	toField := c.Delivery.ToDataField
	toVals := datum.StringDefault(toField, "")
	if toVals == "" {
		return ActContinueError, fmt.Errorf("destination address not found/field `%s` is empty", toField)
	}
	toList, err := stdmail.ParseAddressList(toVals)
	if err != nil {
		return ActContinueError, err
	}

	// setup body
	if c.Delivery.MailFormat == HtmlFormat {
		msg.SetBody(mail.TextHTML, body)
	} else {
		msg.SetBody(mail.TextPlain, body)
	}

	// setup subject field
	subjectField := c.Delivery.SubjectDataField
	subject := datum.StringDefault(subjectField, c.Delivery.DefaultSubject)
	msg.SetSubject(subject)

	// setup from
	msg.SetFrom(c.Delivery.From)

	// set destination
	dest := ""
	toCount := 0
	if c.Delivery.SendMode {
		for _, to := range toList {
			if c.Delivery.SkipIfSent && m.mailSent(to.Address) {
				// skip already send email
				m.ui.Logf("Skipping address: %s, email already sent\n", to.Address)
				continue
			}
			fmt.Fprintln(m.sentWr, to.Address)
			toCount++

			msg.AddTo(to.String())
			if dest != "" {
				dest += ";"
			}
			dest += to.String()
		}
		for _, cc := range m.ccList {
			msg.AddCc(cc)
		}
		for _, bcc := range m.bccList {
			msg.AddBcc(bcc)
		}
		if toCount == 0 {
			return ActSend, nil
		}
	} else {
		// Test address
		dest = c.Delivery.TestAddress
		msg.AddTo(c.Delivery.TestAddress)
	}

	// Ask for confirmation
	if !c.Delivery.SkipConfirmBeforeSend {
		str := fmt.Sprintf("Send email to %s [(Y)es/(N)o/Yes to (A)ll/(C)ancel?", dest)
		action, err := m.ui.Confirm(str)
		if err != nil {
			return action, err
		}
	}

	m.ui.Logf("Sending email to: %s\n", dest)
	if err := msg.Send(conn); err != nil {
		return ActContinueError, err
	}
	return ActContinueError, msg.GetError()
}
