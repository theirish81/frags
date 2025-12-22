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

import "github.com/blues/jsonata-go"

type Transformer struct {
	Name             string  `yaml:"name" json:"name"`
	OnFunctionOutput *string `yaml:"onFunctionOutput" json:"on_function_output"`
	Jsonata          *string `yaml:"jsonata" json:"jsonata"`
	Code             *string `yaml:"code" json:"code"`
}

type Transformers []Transformer

func (t Transformers) FilterOnFunctionOutput(name string) Transformers {
	t2 := make(Transformers, 0)
	for _, t := range t {
		if t.OnFunctionOutput != nil && *t.OnFunctionOutput == name {
			t2 = append(t2, t)
		}
	}
	return t2
}

func (t Transformer) Transform(data map[string]any, runner ExportableRunner) (map[string]any, error) {
	if t.Jsonata != nil {
		script, err := jsonata.Compile(*t.Jsonata)
		if err != nil {
			return data, err
		}
		res, err := script.Eval(data)
		if err != nil {
			return data, err
		}
		if typed, ok := res.(map[string]any); ok {
			return typed, nil
		}
		return map[string]any{"result": res}, nil
	}
	if runner.ScriptEngine() != nil && t.Code != nil {
		return runner.ScriptEngine().RunCode(*t.Code, data, runner)
	}
	return data, nil
}

func (t Transformers) Transform(data map[string]any, runner ExportableRunner) (map[string]any, error) {
	tmp := data
	var err error
	for _, tx := range t {
		tmp, err = tx.Transform(tmp, runner)
		if err != nil {
			return tmp, err
		}
	}
	return tmp, nil
}
