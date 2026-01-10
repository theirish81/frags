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
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/jinzhu/copier"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (m McpServerConfigs) McpTools() McpTools {
	tools := make(McpTools, 0)
	for name, server := range m {
		if !server.Disabled {
			tools = append(tools, NewMcpTool(name, server))
		}
	}
	return tools
}

// McpTool is a wrapper around the MCP client
type McpTool struct {
	Name         string
	serverConfig McpServerConfig
	client       *mcp.Client
	session      *mcp.ClientSession
	log          *slog.Logger
}

// NewMcpTool creates a new MCP client wrapper
func NewMcpTool(name string, serverConfig McpServerConfig) *McpTool {
	client := mcp.NewClient(&mcp.Implementation{Name: name, Version: "v1.0.0"}, nil)
	return &McpTool{
		Name:         name,
		serverConfig: serverConfig,
		client:       client,
		log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

// Connect connects to the MCP server
func (c *McpTool) Connect(ctx context.Context) error {
	if len(c.serverConfig.Command) > 0 {
		return c.ConnectStd(ctx)
	}
	if c.serverConfig.Transport == "sse" {
		return c.ConnectSSE(ctx)
	}
	return c.ConnectStreamableHttp(ctx)
}

// ConnectStd connects to the MCP server using a std/stdout transport
func (c *McpTool) ConnectStd(ctx context.Context) error {
	var err error
	cmd := exec.Command(c.serverConfig.Command, c.serverConfig.Args...)
	cmd.Env = make([]string, 0)
	for k, v := range c.serverConfig.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if len(c.serverConfig.Cwd) > 0 {
		cmd.Dir = c.serverConfig.Cwd
	}
	c.session, err = c.client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	return err
}

// ConnectSSE connects to the MCP server using an SSE transport
func (c *McpTool) ConnectSSE(ctx context.Context) error {
	client := http.Client{
		Timeout:   30 * time.Minute,
		Transport: NewMcpTransport(c.serverConfig.Headers),
	}
	var err error
	c.session, err = c.client.Connect(ctx, &mcp.SSEClientTransport{
		Endpoint:   c.serverConfig.Url,
		HTTPClient: &client,
	}, nil)
	return err
}

// ConnectStreamableHttp connects to the MCP server using a Streamable HTTP transport, which is now the default.
// In case of failure, it falls back to SSE transport
func (c *McpTool) ConnectStreamableHttp(ctx context.Context) error {
	client := http.Client{
		Timeout:   30 * time.Minute,
		Transport: NewMcpTransport(c.serverConfig.Headers),
	}
	var err error
	c.session, err = c.client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   c.serverConfig.Url,
		HTTPClient: &client,
	}, nil)
	if err != nil {
		c.log.Warn("Streamable HTTP transport failed, falling back to SSE transport", "error", err.Error())
		return c.ConnectSSE(ctx)
	}
	return err
}

// ListTools lists the tools available on the server
func (c *McpTool) ListTools(ctx context.Context) (ToolDefinitions, error) {
	res := ToolDefinitions{}
	tools, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return res, err
	}
	// Unfortunately the library seems to return InputSchema in multiple, bizzarre ways, so we need to make sure
	// we convert it into something predictable.
	for _, t := range tools.Tools {
		schema := Schema{}
		switch typed := t.InputSchema.(type) {
		case string:
			if err := json.Unmarshal([]byte(typed), &schema); err != nil {
				return res, err
			}
		case map[string]any:
			if err := mapstructure.Decode(typed, &schema); err != nil {
				return res, err
			}
		case []byte:
			if err := json.Unmarshal(typed, &schema); err != nil {
				return res, err
			}
		case json.RawMessage:
			if err := json.Unmarshal(typed, &schema); err != nil {
				return res, err
			}
		default:
			if err := copier.Copy(&schema, typed); err != nil {
				return res, err
			}
		}
		sPointer := &schema
		if schema.Properties == nil {
			sPointer = nil
		}
		res = append(res, ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: sPointer,
		})
	}
	return res, nil
}

// AsFunctions returns the tools as functions
func (c *McpTool) AsFunctions(ctx context.Context) (Functions, error) {
	functions := Functions{}
	tools, err := c.ListTools(ctx)
	if err != nil {
		return functions, err
	}
	for _, t := range tools {
		functions[t.Name] = Function{
			Name:        t.Name,
			Description: t.Description,
			Collection:  c.Name,
			Schema:      t.InputSchema,
			Func: func(data map[string]any) (any, error) {
				return c.Run(context.Background(), t.Name, data)
			},
		}
	}
	return functions, nil
}

// Run runs a tool on the server
func (c *McpTool) Run(ctx context.Context, name string, arguments any) (any, error) {
	res, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return nil, err
	}
	if res.StructuredContent != nil {
		return res.StructuredContent, nil
	}

	return convertContentArray(res.Content), nil
}

func (c *McpTool) Close() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}

type McpTools []*McpTool

func (m McpTools) Connect(ctx context.Context) error {
	for _, t := range m {
		if err := t.Connect(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m McpTools) Close() error {
	for _, t := range m {
		if err := t.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (m McpTools) AsFunctions(ctx context.Context) (Functions, error) {
	functions := Functions{}
	for _, t := range m {
		fs, err := t.AsFunctions(ctx)
		if err != nil {
			return nil, err
		}
		for k, v := range fs {
			functions[k] = v
		}
	}
	return functions, nil
}

// convertContentArray deals with the fact that the returned content, when it's not explicitly structured, can be a
// bit of an odd ball. If items in the array are of type mcp.TextContent,we will try to convert them into a map[string]any,
// or slices, otherwise plain text is fine. If the array is made of one single item, we return that item directly.
func convertContentArray(content []mcp.Content) any {
	stage1 := make([]any, 0)
	for _, c := range content {
		if textContent, ok := c.(*mcp.TextContent); ok {
			stage1 = append(stage1, convertTextContent(textContent))
		}
	}
	if len(stage1) == 1 {
		return stage1[0]
	}
	return stage1
}

// convertTextContent tries to convert its textual content into a map[string]any or a slice[any]. If both fail, it
// returns the text as a string.
func convertTextContent(content *mcp.TextContent) any {
	if content == nil {
		return ""
	}
	theMap := make(map[string]any)
	if err := json.Unmarshal([]byte(content.Text), &theMap); err == nil {
		return theMap
	}
	slice := make([]any, 0)
	if err := json.Unmarshal([]byte(content.Text), &slice); err == nil {
		return slice
	}
	return content.Text
}

// McpTransport is a wrapper around the default http.RoundTripper that adds default headers to every request
type McpTransport struct {
	Base    http.RoundTripper
	Headers map[string]string
}

func NewMcpTransport(headers map[string]string) *McpTransport {
	return &McpTransport{
		Headers: headers,
		Base: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// RoundTrip adds default headers to the request
func (t *McpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())

	// Add your default headers
	for key, value := range t.Headers {
		req2.Header.Set(key, value)
	}
	return t.Base.RoundTrip(req2)
}
