package inventory

import (
	"bytes"
	"strconv"
	"strings"
	"text/template"
)

func TemplateToString(tmpl *template.Template, data interface{}) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

func joinInt(s []int, prefix, sep string) string {
	parts := make([]string, len(s))
	for i, s := range s {
		parts[i] = prefix + strconv.Itoa(s)
	}
	return strings.Join(parts, sep)
}

func joinStr(s []string, prefix, sep string) string {
	parts := make([]string, len(s))
	for i, s := range s {
		parts[i] = prefix + s
	}
	return strings.Join(parts, sep)
}

func NewTemplate(name, content string) *template.Template {
	tmpl, err := template.New(name).Funcs(
		template.FuncMap{"joinInt": joinInt, "joinStr": joinStr}).Parse(content)
	if err != nil {
		panic(err)
	}
	return tmpl
}
