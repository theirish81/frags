package frags

import (
	"bytes"
	"errors"
	"strings"
	"text/template"

	"github.com/expr-lang/expr"
)

const (
	paramsAttr     = "params"
	contextAttr    = "context"
	componentsAttr = "components"
	iteratorAttr   = "it"
	varsAttr       = "vars"
)

type EvalScope map[string]any

func (e EvalScope) WithIterator(it any) EvalScope {
	e[iteratorAttr] = it
	return e
}

func (e EvalScope) WithVars(vars map[string]any) EvalScope {
	if vars == nil {
		e[varsAttr] = make(map[string]any)
	} else {
		e[varsAttr] = vars
	}
	return e
}

// newEvalScope returns a new scope for evaluating expressions.
func (r *Runner[T]) newEvalScope() EvalScope {
	return EvalScope{
		paramsAttr:     r.params,
		contextAttr:    *r.dataStructure,
		componentsAttr: r.sessionManager.Components,
		varsAttr:       make(map[string]any),
		iteratorAttr:   nil,
	}
}

// EvaluateTemplate evaluates a Golang template with the given scope.
func EvaluateTemplate(text string, scope EvalScope) (string, error) {
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
		err = parsedTmpl.Execute(writer, map[string]any(scope))
		if err != nil {
			return text, err
		}
		text = writer.String()
	}
	return text, nil
}

// EvaluateBooleanExpression evaluates a boolean expression with the given scope using expr.
func EvaluateBooleanExpression(expression string, scope EvalScope) (bool, error) {
	c, err := expr.Compile(expression, expr.Env(scope))
	if err != nil {
		return false, err
	}
	res, err := expr.Run(c, map[string]any(scope))
	if err != nil {
		return false, err
	}
	if b, ok := res.(bool); ok && b {
		return true, nil
	}
	return false, nil
}

func EvaluateArrayExpression(expression string, scope EvalScope) ([]any, error) {
	c, err := expr.Compile(expression, expr.Env(scope))
	if err != nil {
		return nil, err
	}
	res, err := expr.Run(c, map[string]any(scope))
	if err != nil {
		return nil, err
	}
	if a, ok := res.([]any); ok {
		return a, nil
	} else {
		return nil, errors.New("expression did not evaluate to an array")
	}
}
