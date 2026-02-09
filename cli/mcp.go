package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/labstack/echo/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/schema"
	"github.com/theirish81/frags/util"
)

type planDef struct {
	Name       string           `json:"name"`
	Parameters frags.Parameters `json:"parameters"`
}

func initMCP(e *echo.Echo) {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "Frags", Version: version}, nil)
	mcp.AddTool(mcpServer, toolPlanSearch,
		func(ctx context.Context, request *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
			files, err := os.ReadDir(rootDir)
			if err != nil {
				return nil, nil, err
			}
			plans := make([]planDef, 0)
			for _, entry := range files {
				if filepath.Ext(entry.Name()) != ".yaml" {
					continue
				}
				data, err := os.ReadFile(filepath.Join(rootDir, entry.Name()))
				if err != nil {
					continue
				}
				sm := frags.NewSessionManager()
				if err = sm.FromYAML(data); err != nil {
					continue
				}
				pm := frags.Parameters{}
				if sm.Parameters != nil {
					pm = sm.Parameters.Parameters
				}
				plans = append(plans, planDef{Name: entry.Name(), Parameters: pm})
			}
			return toCallResult(plans, "plans", nil), nil, nil
		})
	mcp.AddTool(mcpServer, toolPlanRun,
		func(ctx context.Context, request *mcp.CallToolRequest, args runPlanParams) (*mcp.CallToolResult, any, error) {
			data, err := os.ReadFile(filepath.Join(rootDir, args.Name))
			if err != nil {
				return nil, nil, err
			}
			sm := frags.NewSessionManager()
			if err = sm.FromYAML(data); err != nil {
				return nil, nil, err
			}
			toolsConfig, err := readToolsFile()
			if err != nil {
				return nil, nil, err
			}

			res, err := execute(util.WithFragsContext(ctx, 10*time.Minute), sm, args.Parameters, toolsConfig, resources.NewDummyResourceLoader(), log.NewStreamerLogger(slog.Default(), nil, log.InfoChannelLevel))
			if err != nil {
				return nil, nil, err
			}
			return toCallResult(res, "result", sm.Schema), nil, nil
		})
	method := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, nil)
	authMiddleware := requireApiKey(apiKey)
	e.Any("/mcp", echo.WrapHandler(authMiddleware(method)))
}

type runPlanParams struct {
	Name       string         `json:"name"`
	Parameters map[string]any `json:"parameters"`
}

func toCallResult(data any, rootObjectName string, sx *schema.Schema) *mcp.CallToolResult {
	output := map[string]any{rootObjectName: data}
	if sx != nil {
		output["response_schema"] = sx
	}
	content, _ := json.Marshal(output)
	return &mcp.CallToolResult{StructuredContent: output, Content: []mcp.Content{
		&mcp.TextContent{
			Text: string(content),
		},
	}}
}

var toolPlanSearch = &mcp.Tool{
	Name:        "frags_list_plans",
	Description: "lists all Frags plans",
	InputSchema: &jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{},
	},
}

var toolPlanRun = &mcp.Tool{
	Name:        "frags_run_plan",
	Description: "runs a frag plan",
	InputSchema: &jsonschema.Schema{
		Type:     schema.Object,
		Required: []string{"name", "parameters"},
		Properties: map[string]*jsonschema.Schema{
			"name": {
				Type: schema.String,
			},
			"parameters": {
				Type:        schema.Object,
				Description: "the plans parameters, if any is required, empty object otherwise.",
				AdditionalProperties: &jsonschema.Schema{
					Type: schema.String,
				},
			},
		},
	},
}

func requireApiKey(key string) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("x-api-key") == key {
				handler.ServeHTTP(w, r)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
		})
	}
}
