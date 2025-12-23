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

type ToolType string

const (
	ToolTypeInternetSearch ToolType = "internet_search"
	ToolTypeFunction       ToolType = "function"
	ToolTypeMCP            ToolType = "mcp"
)

// Tool defines a tool that can be used in a session.
// Name is either the tool name of the function name
// ServerName is only used for MCP tools
// Description is the tool description. Optional, as the tool should already have a description, fill if you wish
// to override the default
// Type is either internet_search or function
// InputSchema defines the input schema for the tool. When used in a session, it is meant to work as an override of
// the default value provided by MCP or the developer.
type Tool struct {
	Name        string   `json:"name" yaml:"name"`
	ServerName  string   `json:"server_name" yaml:"serverName"`
	Description string   `json:"description" yaml:"description"`
	Type        ToolType `json:"type" yaml:"type"`
	InputSchema *Schema  `json:"input_schema" yaml:"inputSchema"`
}

func (t Tool) String() string {
	switch t.Type {
	case ToolTypeInternetSearch:
		return string(ToolTypeInternetSearch)
	case ToolTypeFunction:
		return fmt.Sprintf("%s/%s", t.Type, t.Name)
	case ToolTypeMCP:
		return fmt.Sprintf("%s/%s", t.Type, t.ServerName)
	}
	return ""
}

type Tools []Tool

// HasType returns true if the tool list contains a tool of the given type. This is useful for "special" tools like
// internet_search, in which the type is all it needs.
func (t *Tools) HasType(tt ToolType) bool {
	for _, tool := range *t {
		if tool.Type == tt {
			return true
		}
	}
	return false
}

// Function represents a function that can be called by the AI model.
type Function struct {
	Func        func(data map[string]any) (map[string]any, error)
	Name        string
	Server      string
	Description string
	Schema      *Schema
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

// Get returns a function by name.
func (f Functions) Get(name string) Function {
	return f[name]
}

// ListByServer returns a subset of functions, filtered by (MCP) server.
func (f Functions) ListByServer(server string) Functions {
	out := Functions{}
	for k, v := range f {
		if v.Server == server {
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
			c.Args, err = EvaluateArgsTemplates(c.Args, r.newEvalScope().WithVars(session.Vars))
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
