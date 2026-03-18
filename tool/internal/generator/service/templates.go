package service

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/ssgohq/ssgo/internal/util/naming"
	"github.com/ssgohq/ssgo/tool/internal/generator/templates"
)

// newTemplates parses all service templates and returns the template set.
func newTemplates() *template.Template {
	funcMap := template.FuncMap{
		"ToSnakeCase":  naming.ToSnakeCase,
		"ToCamelCase":  naming.ToCamelCase,
		"ToPascalCase": naming.ToPascalCase,
		"ToKebabCase":  naming.ToKebabCase,
		"lower":        strings.ToLower,
		"upper":        strings.ToUpper,
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templates.ServiceTemplates, "service/*.tpl")
	if err != nil {
		panic(fmt.Sprintf("failed to parse service templates: %v", err))
	}
	return tmpl
}
