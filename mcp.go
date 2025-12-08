package frags

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"

	"github.com/go-viper/mapstructure/v2"
	"github.com/jinzhu/copier"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// McpConfig defines the configuration for the MCP clients
type McpConfig struct {
	McpServers map[string]McpServerConfig `json:"mcpServers"`
}

// McpServerConfig defines the configuration to connect to a MCP server
type McpServerConfig struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	Cwd       string            `json:"cwd"`
	Transport string            `json:"transport"`
	Url       string            `json:"url"`
}

// McpTool is a wrapper around the MCP client
type McpTool struct {
	Name    string
	client  *mcp.Client
	session *mcp.ClientSession
}

// NewMcpTool creates a new MCP client wrapper
func NewMcpTool(name string) McpTool {
	client := mcp.NewClient(&mcp.Implementation{Name: name, Version: "v1.0.0"}, nil)
	return McpTool{
		Name:   name,
		client: client,
	}
}

// Connect connects to the MCP server
func (c *McpTool) Connect(ctx context.Context, server McpServerConfig) error {
	if len(server.Command) > 0 {
		return c.ConnectStd(ctx, server)
	}
	return c.ConnectSSE(ctx, server)
}

// ConnectStd connects to the MCP server using a std/stdout transport
func (c *McpTool) ConnectStd(ctx context.Context, server McpServerConfig) error {
	var err error
	cmd := exec.Command(server.Command, server.Args...)
	cmd.Env = make([]string, 0)
	for k, v := range server.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if len(server.Cwd) > 0 {
		cmd.Dir = server.Cwd
	}
	c.session, err = c.client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	return err
}

// ConnectSSE connects to the MCP server using an SSE transport
func (c *McpTool) ConnectSSE(ctx context.Context, server McpServerConfig) error {
	var err error
	c.session, err = c.client.Connect(ctx, &mcp.SSEClientTransport{
		Endpoint:   server.Url,
		HTTPClient: http.DefaultClient,
	}, nil)
	return err
}

// ListTools lists the tools available on the server
func (c *McpTool) ListTools(ctx context.Context) (Tools, error) {
	res := Tools{}
	tools, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return res, err
	}
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
		res = append(res, Tool{
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
			Description: t.Description,
			Server:      c.Name,
			Schema:      t.InputSchema,
			Func: func(data map[string]any) (map[string]any, error) {
				res, err := c.Run(context.Background(), t.Name, data)
				if err != nil {
					return nil, err
				}
				// here we ALWAYS return a map, regardless of the type of the result
				switch t := res.(type) {
				case map[string]any:
					return t, nil
				default:
					return map[string]any{"result": res}, nil
				}
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

	if typed, ok := res.Content[0].(*mcp.TextContent); ok {
		return convertTextContent(typed), nil
	}

	return res.Content[0], nil
}

func (c *McpTool) Close() error {
	return c.session.Close()
}

// convertTextContent tries to convert its textual content into a map[string]any or a slice[any]
func convertTextContent(content *mcp.TextContent) any {
	out := make(map[string]any)
	if content == nil {
		return out
	}
	if err := json.Unmarshal([]byte(content.Text), &out); err == nil {
		return out
	}
	slice := make([]any, 0)
	if err := mapstructure.Decode([]byte(content.Text), &slice); err == nil {
		return slice
	}
	_ = mapstructure.Decode(content, &out)
	return out
}
