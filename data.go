package sendme

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// MailData to-be applied to template
type MailData map[string]interface{}

// MailDataCollection stores list of mail data from spreadsheet
type MailDataCollection struct {
	Data []MailData
}

// StringDefault return string value or default
func (m MailData) StringDefault(key, def string) string {
	v, ok := m[key]
	if !ok {
		return def
	}
	switch vv := v.(type) {
	case string:
		return vv
	case *string:
		if vv != nil {
			return *vv
		}
		return ""
	case fmt.Stringer:
		return vv.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// NewMailDataCollection create mail data collection from files
func NewMailDataCollection(conf *Config) (*MailDataCollection, error) {
	mc := MailDataCollection{}
	if conf == nil || conf.Delivery == nil || conf.Delivery.DataFile == "" {
		// silently return default data
		return &mc, nil
	}
	if err := mc.load(conf.Delivery.DataFile); err != nil {
		return nil, err
	}

	return &mc, nil
}

func (m *MailDataCollection) load(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".xlsx":
		return m.loadXlsx(filename)
	case ".csv":
		return m.loadCsv(filename)
	}

	return fmt.Errorf("unknown file type: %s", filename)
}
func (m *MailDataCollection) loadXlsx(filename string) error {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Get first non empty row
	sheet := "Sheet1"
	if idx := f.GetActiveSheetIndex(); idx > 0 {
		sheet = f.GetSheetName(idx)
	} else {
		sheets := f.GetSheetList()
		for _, name := range sheets {
			sheet = name
			break
		}
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return err
	}
	return m.rowsToCollection(rows)
}
func (m *MailDataCollection) loadCsv(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	rd := csv.NewReader(f)
	rd.FieldsPerRecord = -1
	rd.TrimLeadingSpace = true
	rows, err := rd.ReadAll()
	if err != nil {
		return err
	}
	return m.rowsToCollection(rows)
}

func (m *MailDataCollection) rowsToCollection(rows [][]string) error {
	firstRow := -1
	firstCol := -1
	var tags []string
	for r, row := range rows {
		for c, col := range row {
			if col != "" {
				tags = row[c:]
				firstRow = r + 1
				firstCol = c
				break
			}
		}
		if tags != nil {
			for i, tag := range tags {
				tags[i] = strings.TrimSpace(tag)
			}
			break
		}
	}

	if firstRow < 0 || firstCol < 0 {
		return errors.New("non-empty row/col not found")
	}

	ntags := len(tags)
	for r := firstRow; r < len(rows); r++ {
		row := rows[r]
		ncol := len(row)
		if ntags < ncol {
			ncol = ntags
		}

		md := make(MailData)
		for c := firstCol; c < ncol; c++ {
			key := tags[c-firstCol]
			val := strings.TrimSpace(row[c])
			md[key] = val
		}
		m.Data = append(m.Data, md)
	}

	return nil
}
