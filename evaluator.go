package frags

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/expr-lang/expr"
)

// EvaluateTemplate evaluates a Golang template with the given scope.
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

// EvaluateBooleanExpression evaluates a boolean expression with the given scope using expr.
func EvaluateBooleanExpression(expression string, scope any) (bool, error) {
	c, err := expr.Compile(expression, expr.Env(scope))
	if err != nil {
		return false, err
	}
	res, err := expr.Run(c, scope)
	if err != nil {
		return false, err
	}
	if b, ok := res.(bool); ok && b {
		return true, nil
	}
	return false, nil
}
