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

package main

import (
	"errors"

	"github.com/dop251/goja"
	"github.com/theirish81/frags"
)

type JavascriptScriptingEngine struct {
}

func NewJavascriptScriptingEngine() *JavascriptScriptingEngine {
	return &JavascriptScriptingEngine{}
}

func (e *JavascriptScriptingEngine) RunCode(code string, params map[string]any, runner frags.ExportableRunner) (map[string]any, error) {
	vm := goja.New()
	if err := vm.Set("args", params); err != nil {
		return nil, err
	}
	if err := vm.Set("runFunction", func(name string, args map[string]any) map[string]any {
		res, _ := runner.RunFunction(name, args)
		return res
	}); err != nil {
		return nil, err
	}
	res, err := vm.RunString(code)
	if err != nil {
		return nil, err
	}
	if mapOut, ok := res.Export().(map[string]any); ok {
		return mapOut, nil
	} else {
		return nil, errors.New("invalid output from script: " + res.String() + "")
	}
}
