package sendme

import (
	"errors"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// ParseTextTemplates parse templates in text format
func ParseTextTemplates(conf *Config) (Executer, error) {
	if conf == nil || conf.Delivery == nil || len(conf.Delivery.TemplateFiles) == 0 {
		return nil, errors.New("template file(s) not specified")
	}
	filenames := conf.Delivery.TemplateFiles
	tpl := template.New(conf.Delivery.TemplateName).Funcs(sprig.FuncMap())
	return tpl.ParseFiles(filenames...)
}
