package sendme

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

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
	conf       *Config
	data       *MailDataCollection
	server     *mail.SMTPServer
	ccList     []string
	bccList    []string
	tpl        Executer
	ui         Ui
	sentWr     io.Writer
	sentList   []string
	resendList []string
	intBetween time.Duration
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
	if cclist := conf.Delivery.CcList; cclist != "" {
		cc, err := ParseAddressList(cclist)
		if err != nil {
			return nil, fmt.Errorf("parse CC-list %s error: %w", cclist, err)
		}
		for _, addr := range cc {
			m.ccList = append(m.ccList, addr.String())
		}
	}

	if bcclist := conf.Delivery.BccList; bcclist != "" {
		bcc, err := ParseAddressList(bcclist)
		if err != nil {
			return nil, fmt.Errorf("parse BCC-list %s error: %w", bcclist, err)
		}
		for _, addr := range bcc {
			m.bccList = append(m.bccList, addr.String())
		}
	}

	if err := m.readSentList(); err != nil {
		return nil, err
	}

	// default interval between sending email
	m.intBetween, err = time.ParseDuration(conf.Delivery.IntervalBetweenSend)
	if err != nil {
		m.intBetween = time.Second
	}

	return &m, nil
}

func (m *Mailer) readLines(filename string, fn func(string) string) ([]string, error) {
	fd, err := os.Open(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open file %s error: %w", filename, err)
	}
	defer fd.Close()

	lines := []string{}
	scan := bufio.NewScanner(fd)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line != "" {
			lines = append(lines, fn(line))
		}
	}

	return lines, nil
}

func (m *Mailer) readSentList() error {
	/*
		fd, err := os.Open(m.conf.Delivery.SentFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("open file %s error: %w", m.conf.Delivery.SentFile, err)
		}
		defer fd.Close()

		scan := bufio.NewScanner(fd)
		for scan.Scan() {
			addr := strings.TrimSpace(scan.Text())
			if addr != "" {
				m.sentList = append(m.sentList, strings.ToLower(addr))
			}
		}
	*/

	var err error
	m.sentList, err = m.readLines(m.conf.Delivery.SentFile, strings.ToLower)
	if err != nil {
		return err
	}
	sort.Strings(m.sentList)

	// log if verbose mode
	if m.conf.Verbose {
		m.ui.Logf("<<Sent addresses>>\n")
		for _, addr := range m.sentList {
			m.ui.Logf("  %s\n", addr)
		}
	}

	// 2. Read resend
	m.resendList, err = m.readLines(m.conf.Delivery.ResendFile, strings.ToLower)
	if err != nil {
		m.ui.Logf("[WARN] reading resend-list failed: %v\n", err)
	}
	sort.Strings(m.resendList)

	// log if verbose mode
	if m.conf.Verbose {
		m.ui.Logf("<<ReSend addresses>>\n")
		for _, addr := range m.resendList {
			m.ui.Logf("  %s\n", addr)
		}
	}

	return nil
}

func (m *Mailer) mailSent(addr string) bool {
	addr = strings.ToLower(strings.TrimSpace(addr))
	_, resend := sort.Find(len(m.resendList), func(i int) int {
		return strings.Compare(addr, m.resendList[i])
	})
	if resend {
		// force resend email
		return false
	}

	// check if already sent
	_, sent := sort.Find(len(m.sentList), func(i int) int {
		return strings.Compare(addr, m.sentList[i])
	})

	return sent
}

func (m *Mailer) Send(ctx context.Context) (Stats, error) {
	st := Stats{
		Total: len(m.data.Data),
	}
	// ensure connection keep alive
	m.server.KeepAlive = true

	// open connection
	conn, err := m.server.Connect()
	if err != nil {
		return st, fmt.Errorf("connect to smtp server error: %w", err)
	}
	defer conn.Close()

	// load send list
	fd, err := os.OpenFile(m.conf.Delivery.SentFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return st, fmt.Errorf("error opening sent file %s: %w", m.conf.Delivery.SentFile, err)
	}
	defer fd.Close()
	m.sentWr = fd

	// loop through message and send email
	for _, datum := range m.data.Data {
		if !datum.HasFields(m.conf.Delivery.RequiredFields) {
			js, _ := json.Marshal(datum)
			m.ui.Logf("[WARN] Skip DATUM>> %s\n", string(js))
			st.NumSkip++
			continue
		}
		var sb strings.Builder
		if err := m.tpl.Execute(&sb, datum); err != nil {
			js, _ := json.Marshal(datum)
			m.ui.Logf("[WARN] DATUM>> %s\n", string(js))
			st.NumError++
			return st, err
		}

		// send each mail
		action, err := m.sendMail(ctx, conn, datum, sb.String(), &st)
		if err != nil && action != ActContinueError {
			return st, err
		}

		// Check action/message
		switch action {
		case ActAbortSend:
			return st, errors.New("aborted by user")
		case ActContinueError:
			if err != nil {
				m.ui.Logf("Error when sending email: %v\n", err)
			}
		case ActDontSend:
			m.ui.Logf("Skip send by user\n")
		}
	}

	return st, nil
}

func (m *Mailer) sendMail(ctx context.Context, conn *mail.SMTPClient, datum MailData, body string, st *Stats) (int, error) {
	c := m.conf
	msg := mail.NewMSG()

	// get to addr
	toField := c.Delivery.ToDataField
	toVals := datum.StringDefault(toField, "")
	if toVals == "" {
		return ActContinueError, fmt.Errorf("destination address not found/field `%s` is empty", toField)
	}
	toList, err := ParseAddressList(toVals)
	if err != nil {
		return ActContinueError, fmt.Errorf("parse address `%s` error: %w", toVals, err)
	}

	// setup body
	if c.Delivery.MailFormat == HtmlFormat {
		msg.SetBody(mail.TextHTML, body)
	} else {
		msg.SetBody(mail.TextPlain, body)
	}

	// setup attachments
	files := datum.AttachmentFiles()
	for _, af := range files {
		fi := mail.File{
			FilePath: af.FilePath,
			Name:     af.Name,
			Inline:   af.Inline,
		}
		msg.Attach(&fi)
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
	var sbSent strings.Builder
	if c.Delivery.SendMode {
		for _, to := range toList {
			if c.Delivery.SkipIfSent && m.mailSent(to.Address) {
				// skip already send email
				m.ui.Logf("Skipping address: %s, email already sent\n", to.Address)
				st.NumAlreadySent++
				continue
			}
			fmt.Fprintln(&sbSent, to.Address)
			toCount++

			msg.AddTo(to.String())
			if dest != "" {
				dest += ","
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
		str := fmt.Sprintf("Send email to %s [(Y)es/(N)o/Yes to (A)ll/(C)ancel]? ", dest)
		action, err := m.ui.Confirm(str)
		if err != nil {
			return action, fmt.Errorf("user confirmation error %w", err)
		}
		if action == ActSendAll {
			m.ui.Logf("Skip further confirmation\n")
			m.conf.Delivery.SkipConfirmBeforeSend = true
		}

		// check testing
		if action != ActSend && action != ActSendAll {
			return action, nil
		}
	}

	if m.intBetween > 0 {
		time.Sleep(m.intBetween)
	}
	if err := msg.Send(conn); err != nil {
		st.NumError++
		return ActContinueError, fmt.Errorf("sending email to %s error: %w", dest, err)
	}
	m.ui.Logf("Sent email to: %s\n", dest)
	fmt.Fprint(m.sentWr, sbSent.String())
	st.NumSentAddr += toCount
	st.NumSentData++

	return ActContinueError, msg.GetError()
}
