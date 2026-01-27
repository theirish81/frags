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
	"github.com/blues/jsonata-go"
	"github.com/jmespath/go-jmespath"
	"github.com/theirish81/frags/evaluators"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/util"
)

type Parser string

const (
	JsonParser Parser = "json"
	CsvParser  Parser = "csv"
)

// Transformer is a functionality that given a certain input, transforms it into another output using either a
// Jsonata expression or a custom script (if the scripting engine is available). The transformer will run on specific
// triggers. We currently support only OnFunctionOutput.
type Transformer struct {
	Name             string  `yaml:"name" json:"name"`
	OnFunctionInput  *string `yaml:"onFunctionInput,omitempty" json:"onFunctionInput,omitempty"`
	OnFunctionOutput *string `yaml:"onFunctionOutput,omitempty" json:"onFunctionOutput,omitempty"`
	OnResource       *string `yaml:"onResource,omitempty" json:"onResource,omitempty"`
	Jsonata          *string `yaml:"jsonata" json:"jsonata"`
	JmesPath         *string `yaml:"jmesPath" json:"jmesPath"`
	Expr             *string `yaml:"expr" json:"expr"`
	Parser           *Parser `yaml:"parser" json:"parser"`
	Code             *string `yaml:"code" json:"code"`
}

type Transformers []Transformer

// FilterOnFunctionOutput filters the transformers based on the OnFunctionOutput trigger
func (t Transformers) FilterOnFunctionOutput(name string) Transformers {
	t2 := make(Transformers, 0)
	for _, t := range t {
		if t.OnFunctionOutput != nil && *t.OnFunctionOutput == name {
			t2 = append(t2, t)
		}
	}
	return t2
}

func (t Transformers) FilterOnFunctionInput(name string) Transformers {
	t2 := make(Transformers, 0)
	for _, t := range t {
		if t.OnFunctionInput != nil && *t.OnFunctionInput == name {
			t2 = append(t2, t)
		}
	}
	return t2
}

func (t Transformers) FilterOnResource(name string) Transformers {
	t2 := make(Transformers, 0)
	for _, t := range t {
		if t.OnResource != nil && *t.OnResource == name {
			t2 = append(t2, t)
		}
	}
	return t2
}

// Transform applies the transformation to the given data
func (t Transformer) Transform(ctx *util.FragsContext, data any, runner ExportableRunner) (any, error) {
	runner.Logger().Debug(log.NewEvent(log.StartEventType, log.TransformerComponent).WithTransformer(t.Name))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// If a parser is configured, then we try to parse whatever is in data as a JSON or as a CSV. If we fail, then
	// we're done and we bail out
	if t.Parser != nil {
		switch *t.Parser {
		case JsonParser:
			var err error
			data, err = util.ParseJSON(data)
			if err != nil {
				return util.EmptyMap, err
			}
		case CsvParser:
			var err error
			data, err = util.ParseCSV(data)
			if err != nil {
				return util.EmptyMap, err
			}
		}
	}
	// if JSONata is configured...
	if t.Jsonata != nil {
		script, err := jsonata.Compile(*t.Jsonata)
		if err != nil {
			return util.EmptyMap, err
		}
		data, err = script.Eval(data)
		if err != nil {
			return util.EmptyMap, err
		}
	}
	if t.JmesPath != nil {
		var err error
		data, err = jmespath.Search(*t.JmesPath, data)
		if err != nil {
			return util.EmptyMap, err
		}
	}
	if t.Expr != nil {
		var err error
		data, err = evaluators.EvaluateExpression(*t.Expr, evaluators.EvalScope{"args": data})
		if err != nil {
			return util.EmptyMap, err
		}
	}
	if t.Code != nil {
		var err error
		if data, err = runner.ScriptEngine().RunCode(ctx, *t.Code, data, runner); err != nil {
			return util.EmptyMap, err
		}
	}
	return data, nil
}

// Transform applies all the transformations to the given data
func (t Transformers) Transform(ctx *util.FragsContext, data any, runner ExportableRunner) (any, error) {
	tmp := data
	var err error
	for _, tx := range t {
		tmp, err = tx.Transform(ctx, tmp, runner)
		if err != nil {
			return tmp, err
		}
	}
	return tmp, nil
}
