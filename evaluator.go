package frags

import (
	"bytes"
	"strings"
	"text/template"
)

func EvaluateTemplate(text string, scope any) (string, error) {
	for i := 0; i < 3; i++ {
		if scope == nil || !strings.Contains(text, "{{") {
			return text, nil
		}
		tmpl := template.New("tpl")
		parsedTmpl, err := tmpl.Parse(text)
		if err != nil {
			return text, err
		}
		writer := bytes.NewBufferString("")
		err = parsedTmpl.Execute(writer, scope)
		if err != nil {
			return text, err
		}
		text = writer.String()
	}
	return text, nil

}
