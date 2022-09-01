package sendme

import (
	"errors"
	"html/template"

	"github.com/Masterminds/sprig/v3"
)

// ParseHtmlTemplates parse templates in text format
func ParseHtmlTemplates(conf *Config) (Executer, error) {
	if conf == nil || conf.Delivery == nil || len(conf.Delivery.TemplateFiles) == 0 {
		return nil, errors.New("template file(s) not specified")
	}
	filenames := conf.Delivery.TemplateFiles
	tpl := template.New(conf.Delivery.TemplateName).Funcs(sprig.FuncMap())
	return tpl.ParseFiles(filenames...)
}
