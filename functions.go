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
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/samber/lo"
)

// Function represents a function that can be called by the AI model.
// Name is the function name.
// ToolsCollection is the MCP server or collection that contains the function.
// Description is the function description
// Schema is the input schema for the function.
type Function struct {
	Func        func(data map[string]any) (any, error) `yaml:"-"`
	Name        string                                 `yaml:"name"`
	Collection  string                                 `yaml:"collection"`
	Description string                                 `yaml:"description"`
	Schema      *Schema                                `yaml:"schema"`
}

func (f Function) String() string {
	return fmt.Sprintf("%s:%s", f.Collection, f.Name)
}

// Run runs the function, applying any transformers defined in the runner.
func (f Function) Run(args map[string]any, runner ExportableRunner) (any, error) {
	ax, err := runner.Transformers().FilterOnFunctionInput(f.Name).Transform(maps.Clone(args), runner)
	if err != nil {
		return nil, err
	}
	if !isMapAny(ax) {
		return nil, fmt.Errorf("expected map[string]any, got %T", ax)
	}
	runner.Logger().Debug(NewEvent(StartEventType, FunctionComponent).WithFunction(fmt.Sprintf("%s(%v)", f.Name, ax)))
	ax, err = f.Func(ax.(map[string]any))
	if err != nil {
		return nil, err
	}

	out, err := runner.Transformers().FilterOnFunctionOutput(f.Name).Transform(ax, runner)
	if err != nil {
		return nil, err
	}
	runner.Logger().Debug(NewEvent(EndEventType, FunctionComponent).WithFunction(fmt.Sprintf("%s(%v)", f.Name, ax)).WithContent(out))
	return out, nil
}

// Functions is a map of functions, indexed by name.
type Functions map[string]Function

func (f Functions) String() string {
	vals := make([]string, 0)
	for _, v := range f {
		vals = append(vals, v.String())
	}
	return fmt.Sprintf("%v", vals)
}

// Get returns a function by name.
func (f Functions) Get(name string) Function {
	return f[name]
}

// ListByCollection returns a subset of functions, filtered by MCP server or collection
func (f Functions) ListByCollection(collection string) Functions {
	out := Functions{}
	for k, v := range f {
		if v.Collection == collection {
			out[k] = v
		}
	}
	return out
}

type FunctionCallDestination string

const (
	AiFunctionCallDestination      FunctionCallDestination = "ai"
	VarsFunctionCallDestination    FunctionCallDestination = "vars"
	ContextFunctionCallDestination FunctionCallDestination = "context"
)

// FunctionCall Represents a function invocation. NOTE: description is meant to explain to LLM what the output data is
// about when the function is called by an entity that's not the LLM itself.
// IMPORTANT: if Code is not nil, this will trigger the execution of the scripting engine. If the engine is nil, nothing
// will happen.
type FunctionCall struct {
	Name        string                            `yaml:"name" json:"name"`
	Code        *string                           `yaml:"code" json:"code"`
	Args        map[string]any                    `yaml:"args" json:"args"`
	Description *string                           `yaml:"description" json:"description"`
	In          *FunctionCallDestination          `yaml:"in" json:"in" validate:"omitempty,oneof=ai vars context"`
	Var         *string                           `yaml:"var" json:"var"`
	Func        func(map[string]any) (any, error) `yaml:"-" json:"-"`
}

type FunctionCalls []FunctionCall

func (f FunctionCalls) FilterVarsFunctionCalls() FunctionCalls {
	if f == nil {
		return FunctionCalls{}
	}
	fc := lo.Filter(f, func(fc FunctionCall, index int) bool {
		return fc.In != nil && *fc.In == VarsFunctionCallDestination
	})
	return fc
}

func (f FunctionCalls) FilterAiFunctionCalls() FunctionCalls {
	fc := lo.Filter(f, func(fc FunctionCall, index int) bool {
		return fc.In == nil || *fc.In == AiFunctionCallDestination
	})
	return fc
}

func (f FunctionCalls) FilterContextFunctionCalls() FunctionCalls {
	fc := lo.Filter(f, func(fc FunctionCall, index int) bool {
		return fc.In != nil && *fc.In == ContextFunctionCallDestination
	})
	return fc
}

// RunAllFunctionCalls runs all the function calls in the given collection.
func (r *Runner[T]) RunAllFunctionCalls(ctx context.Context, fc FunctionCalls, scope EvalScope) (map[string]any, error) {
	vx := make(map[string]any)
	for _, c := range fc {
		varName := c.Name
		if c.Var != nil {
			varName = *c.Var
		}
		var err error
		vx[varName], err = r.runFunctionCall(ctx, c, scope)
		scope.WithVars(vx)
		if err != nil {
			return vx, err
		}
	}
	return vx, nil
}

// runFunctionCall runs a FunctionCall object, evaluating the arguments if needed.
func (r *Runner[T]) runFunctionCall(ctx context.Context, fc FunctionCall, scope EvalScope) (any, error) {
	clonedFc := fc
	deadline, ok := ctx.Deadline()
	if ok && time.Now().After(deadline) {
		return nil, ctx.Err()
	}
	var err error
	clonedFc.Args, err = EvaluateMapValues(clonedFc.Args, scope)
	if err != nil {
		return nil, err
	}
	if fc.Func != nil {
		return fc.Func(clonedFc.Args)
	} else if fc.Code != nil {
		return r.ScriptEngine().RunCode(*clonedFc.Code, clonedFc.Args, r)
	} else {
		return r.ai.RunFunction(clonedFc, r)
	}
}

// RunSessionAiPreCallsToTextContext runs the pre-call functions and composes a textual context to be prepended to the
// actual prompt.
func (r *Runner[T]) RunSessionAiPreCallsToTextContext(ctx context.Context, session Session, scope EvalScope) (string, error) {
	preCallsText := ""
	if session.PreCalls != nil {
		for _, c := range session.PreCalls.FilterAiFunctionCalls() {
			res, err := r.runFunctionCall(ctx, c, scope)
			if err != nil {
				return preCallsText, err
			}
			preCallsText += preCallCtx(c, res)
		}
	}
	return preCallsText, nil
}
