/*
 * Copyright (C) 2026 Simone Pezzano
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

package scriptengines

import (
	"time"

	"github.com/dop251/goja"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/util"
)

type JavascriptScriptingEngine struct {
}

func NewJavascriptScriptingEngine() *JavascriptScriptingEngine {
	return &JavascriptScriptingEngine{}
}

func (e *JavascriptScriptingEngine) RunCode(ctx *util.FragsContext, code string, params any, runner frags.ExportableRunner) (any, error) {
	innerCtx := util.WithFragsContext(ctx, 1*time.Minute)
	defer innerCtx.Cancel(nil)
	vm := goja.New()
	go func() {
		<-innerCtx.Done()
		vm.Interrupt("timeout")
	}()
	var args any
	switch t := params.(type) {
	case []byte:
		args = string(t)
	default:
		args = params
	}
	if err := vm.Set("args", args); err != nil {
		return nil, err
	}
	if err := vm.Set("runFunction", func(name string, args map[string]any) any {
		res, err := runner.RunFunction(innerCtx, name, args)
		if err != nil {
			panic(vm.NewTypeError(err.Error()))
		}
		return res
	}); err != nil {
		return nil, err
	}
	res, err := vm.RunString(code)
	if err != nil {
		return nil, err
	}
	return res.Export(), nil
}
