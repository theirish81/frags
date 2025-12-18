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
// Description is the tool description. Optional, as the tool should already have a description, fill if you wish
// to override the default
// Type is either internet_search or function
// InputSchema is used only for functions, and defines the parameters of the function. Optional, as the tool should
// already have parameters, fill if you wish to override the default
type Tool struct {
	Name        string   `json:"name" yaml:"name"`
	ServerName  string   `json:"server_name" yaml:"serverName"`
	Description string   `json:"description" yaml:"description"`
	Type        ToolType `json:"type" yaml:"type"`
	InputSchema *Schema  `json:"inputSchema" yaml:"input_schema"`
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
	Server      string
	Description string
	Schema      *Schema
}

func (f Function) Run(data map[string]any) (map[string]any, error) {
	return f.Func(data)
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

type FunctionCall struct {
	Name        string         `yaml:"name" json:"name"`
	Args        map[string]any `yaml:"args" json:"args"`
	Description *string        `yaml:"description" json:"description"`
}

type FunctionCalls []FunctionCall

// RunPreCallsToTextContext runs the pre-call functions, and composes a textual context to be prepended to the
// actual prompt.
func (r *Runner[T]) RunPreCallsToTextContext(ctx context.Context, session Session) (string, error) {
	preCallsText := ""
	if session.PreCalls != nil {
		for _, c := range *session.PreCalls {
			deadline, _ := ctx.Deadline()
			if time.Now().After(deadline) {
				return preCallsText, ctx.Err()
			}
			res, err := r.ai.RunFunction(c)
			if err != nil {
				return preCallsText, err
			}
			preCallsText += preCallCtx(c, res)
		}
	}
	return preCallsText, nil
}
