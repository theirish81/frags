/*
 * Copyright (C) 2025 Simone Pezzano
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package frags

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
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

// EvalScope is the scope for evaluating expressions.
type EvalScope map[string]any

// WithIterator clones the scope and adds the iterator.
func (e EvalScope) WithIterator(it any) EvalScope {
	e[iteratorAttr] = it
	return e
}

// WithVars clones the scope and adds the vars.
func (e EvalScope) WithVars(vars map[string]any) EvalScope {
	if vars == nil {
		return e
	}

	for k, v := range vars {
		e[varsAttr].(map[string]any)[k] = v
	}
	return e
}

// Vars returns the vars map.
func (e EvalScope) Vars() map[string]any {
	return e[varsAttr].(map[string]any)
}

// NewEvalScope is the EvalScope constructor, unbounded to a specific Runner.
func NewEvalScope() EvalScope {
	return EvalScope{
		paramsAttr:     make(map[string]any),
		contextAttr:    make(map[string]any),
		componentsAttr: make(map[string]any),
		varsAttr:       make(map[string]any),
	}
}

// newEvalScope returns a new scope for evaluating expressions.
func (r *Runner[T]) newEvalScope() EvalScope {
	return EvalScope{
		paramsAttr:     r.params,
		contextAttr:    *r.dataStructure,
		componentsAttr: r.sessionManager.Components,
		varsAttr:       r.vars,
		iteratorAttr:   nil,
	}
}

// EvaluateTemplate evaluates a Golang template with the given scope.
func EvaluateTemplate(text string, scope EvalScope) (string, error) {
	for i := 0; i < 3; i++ {
		if scope == nil || !strings.Contains(text, "{{") {
			return text, nil
		}
		tmpl := template.New("tpl").Funcs(templateFuncs)
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

func EvaluateExpression(expression string, scope EvalScope) (any, error) {
	c, err := expr.Compile(expression, append(exprFunctions(), expr.Env(scope))...)
	if err != nil {
		return nil, err
	}
	return expr.Run(c, map[string]any(scope))
}

// EvaluateBooleanExpression evaluates a boolean expression with the given scope using expr.
func EvaluateBooleanExpression(expression string, scope EvalScope) (bool, error) {
	c, err := expr.Compile(expression, append(exprFunctions(), expr.Env(scope))...)
	if err != nil {
		return false, err
	}
	res, err := expr.Run(c, map[string]any(scope))
	if err != nil {
		return false, err
	}
	if b, ok := res.(bool); ok {
		return b, nil
	}
	return false, errors.New("return type is not a boolean")
}

// EvaluateArrayExpression evaluates an array expression, expecting the target to be an array.
func EvaluateArrayExpression(expression string, scope EvalScope) ([]any, error) {
	c, err := expr.Compile(expression, append(exprFunctions(), expr.Env(scope))...)
	if err != nil {
		return nil, err
	}
	res, err := expr.Run(c, map[string]any(scope))
	if err != nil {
		return nil, err
	}
	rv := toConcreteValue(reflect.ValueOf(res))
	if rv.Kind() == reflect.Slice {
		result := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = rv.Index(i).Interface()
		}
		return result, nil
	} else {
		return nil, errors.New("expression did not evaluate to an array")
	}
}

// EvaluateMapValues will evaluate all **first level** strings as templates in a given map of arguments.
func EvaluateMapValues(args map[string]any, scope EvalScope) (map[string]any, error) {
	if args == nil {
		return nil, nil
	}
	for k, v := range args {
		if s, ok := v.(string); ok {
			res, err := EvaluateTemplate(s, scope)
			if err != nil {
				return nil, err
			}
			args[k] = res
		}
	}
	return args, nil
}

// templateFuncs are the functions available in the templates.
var templateFuncs = template.FuncMap{
	"json": func(v any) string {
		json, _ := json.MarshalIndent(v, "", " ")
		return string(json)
	},
	"kf": func(v any) string {
		return ToKFormat(v)
	},
}

type Vars map[string]any

func (v *Vars) Apply(data map[string]any) {
	for k, val := range data {
		(*v)[k] = val
	}
}

func exprFunctions() []expr.Option {
	return []expr.Option{
		expr.Function("unique",
			func(params ...any) (any, error) {
				seen := make(map[any]bool)
				result := make([]any, 0)
				input, ok := params[0].([]any)
				if !ok {
					return nil, errors.New("unique function expects an array as input")
				}
				for _, item := range input {
					if !seen[item] {
						seen[item] = true
						result = append(result, item)
					}
				}
				return result, nil
			}, new(func([]any) []any)),
		expr.Function("chunk",
			func(params ...any) (any, error) {
				chunks := make([]any, 0)
				input, ok := params[0].([]any)
				if !ok {
					return nil, errors.New("chunk function expects an array as input")
				}
				size, ok := params[1].(int)
				if !ok {
					return nil, errors.New("chunk function expects an integer as second parameter")
				}
				for i := 0; i < len(input); i += size {
					end := i + size
					if end > len(input) {
						end = len(input)
					}
					chunks = append(chunks, input[i:end])
				}
				return chunks, nil
			}, new(func([]any, int) []any)),
	}
}
