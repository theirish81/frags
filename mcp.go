package frags

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"

	"github.com/jinzhu/copier"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type McpConfig struct {
	McpServers map[string]McpServer `json:"mcpServers"`
}

type McpServer struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	Cwd       string            `json:"cwd"`
	Transport string            `json:"transport"`
	Url       string            `json:"url"`
}

type McpTool struct {
	Name    string
	client  *mcp.Client
	session *mcp.ClientSession
}

func NewMcpTool(name string) McpTool {
	client := mcp.NewClient(&mcp.Implementation{Name: name, Version: "v1.0.0"}, nil)
	return McpTool{
		Name:   name,
		client: client,
	}
}

func (c *McpTool) Connect(ctx context.Context, server McpServer) error {
	if len(server.Command) > 0 {
		return c.ConnectStd(ctx, server)
	}
	return c.ConnectSSE(ctx, server)
}

func (c *McpTool) ConnectStd(ctx context.Context, server McpServer) error {
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

func (c *McpTool) ConnectSSE(ctx context.Context, server McpServer) error {
	var err error
	c.session, err = c.client.Connect(ctx, &mcp.SSEClientTransport{
		Endpoint:   server.Url,
		HTTPClient: http.DefaultClient,
	}, nil)
	return err
}

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
			Parameters:  sPointer,
		})
	}
	return res, nil
}

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
			Schema:      t.Parameters,
			Func: func(data map[string]any) (map[string]any, error) {
				res, err := c.Run(context.Background(), t.Name, data)
				if err != nil {
					return nil, err
				}
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

func (c *McpTool) Run(ctx context.Context, name string, arguments any) (any, error) {
	res, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return nil, err
	}
	return res.StructuredContent, nil
}

func (c *McpTool) Close() error {
	return c.session.Close()
}
