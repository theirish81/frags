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
	"time"
)

// Function represents a function that can be called by the AI model.
// Name is the function name.
// ToolsCollection is the MCP server or collection that contains the function.
// Description is the function description
// Schema is the input schema for the function.
type Function struct {
	Func        func(data map[string]any) (map[string]any, error) `yaml:"-"`
	Name        string                                            `yaml:"name"`
	Collection  string                                            `yaml:"collection"`
	Description string                                            `yaml:"description"`
	Schema      *Schema                                           `yaml:"schema"`
}

func (f Function) String() string {
	return fmt.Sprintf("%s:%s", f.Collection, f.Name)
}

// Run runs the function, applying any transformers defined in the runner.
func (f Function) Run(args map[string]any, runner ExportableRunner) (map[string]any, error) {
	args, err := runner.Transformers().FilterOnFunctionInput(f.Name).Transform(args, runner)
	if err != nil {
		return nil, err
	}
	data, err := f.Func(args)
	if err != nil {
		return nil, err
	}
	return runner.Transformers().FilterOnFunctionOutput(f.Name).Transform(data, runner)
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

// FunctionCall Represents a function invocation. NOTE: description is meant to explain to LLM what the output data is
// about when the function is called by an entity that's not the LLM itself.
// IMPORTANT: if Code is not nil, this will trigger the execution of the scripting engine. If the engine is nil, nothing
// will happen.
type FunctionCall struct {
	Name        string         `yaml:"name" json:"name"`
	Code        *string        `yaml:"code" json:"code"`
	Args        map[string]any `yaml:"args" json:"args"`
	Description *string        `yaml:"description" json:"description"`
}

type FunctionCalls []FunctionCall

// RunPreCallsToTextContext runs the pre-call functions and composes a textual context to be prepended to the
// actual prompt.
func (r *Runner[T]) RunPreCallsToTextContext(ctx context.Context, session Session) (string, error) {
	preCallsText := ""
	if session.PreCalls != nil {
		for _, c := range *session.PreCalls {
			deadline, _ := ctx.Deadline()
			if time.Now().After(deadline) {
				return preCallsText, ctx.Err()
			}
			var err error
			c.Args, err = EvaluateMapValues(c.Args, r.newEvalScope().WithVars(r.vars).WithVars(session.Vars))
			if err != nil {
				return preCallsText, err
			}
			if c.Code != nil {
				res, err := r.ScriptEngine().RunCode(*c.Code, c.Args, r)
				if err != nil {
					return preCallsText, err
				}
				preCallsText += preCallCtx(c, res)
			} else {
				res, err := r.ai.RunFunction(c, r)
				if err != nil {
					return preCallsText, err
				}
				preCallsText += preCallCtx(c, res)
			}
		}
	}
	return preCallsText, nil
}
