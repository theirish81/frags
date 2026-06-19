package anthropic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/avast/retry-go/v5"
	"github.com/jinzhu/copier"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/schema"
	"github.com/theirish81/frags/util"
)

const engine = "anthropic"

// Anthropic defaults
const temperature float32 = 0.4
const topK float32 = 0
const topP float32 = 0

const defaultModel = "claude-3-5-sonnet-20241022"

// Ai is a wrapper around the anthropic client for Frags
type Ai struct {
	client       *anthropic.Client
	systemPrompt string
	content      []anthropic.MessageParam
	Functions    frags.ExternalFunctions
	config       Config
}

type Config struct {
	Model       string        `yaml:"model" json:"model"`
	Temperature float32       `yaml:"temperature" json:"temperature"`
	TopK        float32       `yaml:"topK" json:"topK"`
	TopP        float32       `yaml:"topP" json:"topP"`
	MaxTokens   int           `yaml:"maxTokens" json:"maxTokens"`
	Attempts    int           `yaml:"attempts" json:"attempts"`
	RetryDelay  time.Duration `yaml:"retryDelay" json:"retryDelay"`
}

func DefaultConfig() Config {
	return Config{
		Model:       defaultModel,
		Temperature: temperature,
		TopK:        topK,
		TopP:        topP,
	}
}

func (d *Ai) SetSystemPrompt(prompt string) {
	d.systemPrompt = prompt
}

// NewAI creates a new Ai wrapper
func NewAI(client *anthropic.Client, config Config) *Ai {
	return &Ai{
		client:    client,
		content:   make([]anthropic.MessageParam, 0),
		Functions: frags.ExternalFunctions{},
		config:    config,
	}
}

// New creates a new Ai instance and returns it
func (d *Ai) New() frags.Ai {
	return &Ai{
		client:       d.client,
		content:      make([]anthropic.MessageParam, 0),
		Functions:    d.Functions,
		config:       d.config,
		systemPrompt: d.systemPrompt,
	}
}

// Ask performs a query against the Anthropic API, according to the Frags interface
func (d *Ai) Ask(ctx *util.FragsContext, text string, sx *schema.Schema, tools frags.ToolDefinitions,
	runner frags.ExportableRunner, rx ...resources.ResourceData) ([]byte, error) {

	blocks := make([]anthropic.ContentBlockParamUnion, 0)
	for _, resource := range rx {
		runner.Logger().Debug(log.NewEvent(log.LoadEventType, log.AiComponent).WithResource(resource.Identifier).WithEngine(engine))
		if resource.MediaType == util.MediaText || resource.MediaType == "" {
			content := fmt.Sprintf("=== %s ===\n%s\n===\n", resource.Identifier, string(resource.ByteContent))
			blocks = append(blocks, anthropic.NewTextBlock(content))
		} else if strings.HasPrefix(resource.MediaType, "image/") {
			b64 := base64.StdEncoding.EncodeToString(resource.ByteContent)
			blocks = append(blocks, anthropic.NewImageBlockBase64(resource.MediaType, b64))
		} else {
			b64 := base64.StdEncoding.EncodeToString(resource.ByteContent)
			blocks = append(blocks, anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{
				// MediaType: anthropic.F(resource.MediaType),
				Data: b64,
			}))
		}
	}

	blocks = append(blocks, anthropic.NewTextBlock(text))
	newMsg := anthropic.NewUserMessage(blocks...)

	if text == "" && sx == nil && len(d.content) > 0 {
		lastMsg := d.content[len(d.content)-1]
		out := ""
		for _, block := range lastMsg.Content {
			if block.OfText != nil {
				out += block.OfText.Text
			}
		}
		return []byte(out), nil
	}

	tx, err := d.configureTools(tools)
	runner.Logger().Debug(log.NewEvent(log.GenericEventType, log.AiComponent).WithEngine(engine).WithMessage("configured tools").WithContent(tools))
	if err != nil {
		return nil, err
	}

	d.content = append(d.content, newMsg)

	keepGoing := true
	out := ""
	counter := 0

	for keepGoing {
		counter++
		if counter >= 10 {
			return nil, errors.New("loop detected. Too many iterations")
		}
		runner.Logger().Debug(log.NewEvent(log.StartEventType, log.AiComponent).WithMessage("generating content").WithEngine(engine))
		var res *anthropic.Message

		if d.config.Attempts <= 0 {
			d.config.Attempts = 1
		}

		if err = retry.New(retry.Attempts(uint(d.config.Attempts)), retry.Delay(d.config.RetryDelay), retry.Context(ctx),
			retry.DelayType(retry.BackOffDelay), retry.RetryIf(func(err error) bool {
				errStr := err.Error()
				if strings.Contains(errStr, "429") || strings.Contains(errStr, "500") || strings.Contains(errStr, "502") || strings.Contains(errStr, "503") || strings.Contains(errStr, "504") || strings.Contains(errStr, "timeout") {
					return true
				}
				var netErr net.Error
				if errors.As(err, &netErr) {
					return true
				}
				return false
			}), retry.OnRetry(func(attempt uint, err error) {
				runner.Logger().Info(log.NewEvent(log.GenericEventType, log.AiComponent).WithMessage("Anthropic infrastructure is overloaded, retrying...").WithEngine(engine).WithErr(err).WithIteration(int(attempt)))
			})).Do(func() error {

			params := anthropic.MessageNewParams{
				Model:     d.config.Model,
				MaxTokens: int64(d.config.MaxTokens),
				Messages:  d.content,
			}
			if d.config.Temperature > 0 {
				params.Temperature = anthropic.Float(float64(d.config.Temperature))
			}
			if d.config.TopP > 0 {
				params.TopP = anthropic.Float(float64(d.config.TopP))
			}
			if d.config.TopP > 0 {
				params.TopP = anthropic.Float(float64(d.config.TopP))
			}

			if sx != nil {
				params.OutputConfig = anthropic.OutputConfigParam{
					Format: anthropic.JSONOutputFormatParam{
						Schema: SchemaToClaudeMap(sx),
					},
				}
			}

			if d.config.TopK > 0 {
				params.TopK = anthropic.Opt(int64(d.config.TopK))
			}
			if len(d.systemPrompt) > 0 {
				params.System = []anthropic.TextBlockParam{{Text: d.systemPrompt}}
			}
			if len(tx) > 0 {
				params.Tools = tx
			}

			res, err = d.client.Messages.New(ctx, params)
			return err
		}); err != nil {
			return nil, err
		}

		if res == nil || len(res.Content) == 0 {
			keepGoing = false
			continue
		}

		// adding assistant response to history
		assistantParams := make([]anthropic.ContentBlockParamUnion, len(res.Content))
		for i, block := range res.Content {
			assistantParams[i] = block.ToParam()
		}
		d.content = append(d.content, anthropic.NewAssistantMessage(assistantParams...))

		hasToolCalls := false
		var toolResultBlocks []anthropic.ContentBlockParamUnion

		// temporary storage for text in this turn
		currentTurnText := ""

		for _, block := range res.Content {
			if block.Type == "text" {
				currentTurnText += block.Text
			} else if block.Type == "tool_use" {
				hasToolCalls = true
				fres, ferr := d.RunFunction(ctx, frags.FunctionCaller{Name: block.Name, Args: decodeRawJsonMessageToMap(block.Input)}, runner)

				fresBytes, _ := json.Marshal(NewFunctionResponseMap(fres, ferr))
				toolResultBlocks = append(toolResultBlocks, anthropic.NewToolResultBlock(block.ID, string(fresBytes), ferr != nil))
			}
		}

		if hasToolCalls {
			d.content = append(d.content, anthropic.NewUserMessage(toolResultBlocks...))
			keepGoing = true
		} else {
			keepGoing = false
			out = currentTurnText
			runner.Logger().Debug(log.NewEvent(log.EndEventType, log.AiComponent).WithMessage("generated content").WithEngine(engine).WithContent(out))
			if strings.Contains(out, "[FATAL]") {
				err = errors.New(out)
			}
		}
	}
	return []byte(out), err
}

func (d *Ai) configureTools(tools frags.ToolDefinitions) ([]anthropic.ToolUnionParam, error) {
	tx := make([]anthropic.ToolUnionParam, 0)
	for _, tool := range tools {
		switch tool.Type {
		case frags.ToolTypeFunction:
			if fx, found := d.Functions[tool.Name]; found {
				pSchema := fx.Schema
				if tool.InputSchema != nil {
					pSchema = tool.InputSchema
				}

				var inputSchema any
				if pSchema != nil {
					schemaBytes, _ := json.Marshal(pSchema)
					_ = json.Unmarshal(schemaBytes, &inputSchema)
				} else {
					inputSchema = map[string]any{"type": "object", "properties": map[string]any{}}
				}

				description := fx.Description
				if len(tool.Description) > 0 {
					description = tool.Description
				}
				px := anthropic.ToolInputSchemaParam{}
				_ = copier.Copy(&px, inputSchema)
				tx = append(tx, anthropic.ToolUnionParam{
					OfTool: &anthropic.ToolParam{
						Name:        tool.Name,
						Description: anthropic.Opt(description),
						InputSchema: px,
					},
				})
			}
		case frags.ToolTypeMCP, frags.ToolTypeCollection:
			for k, v := range d.Functions.ListByCollection(tool.Name) {
				if tool.Allowlist == nil || slices.Contains(*tool.Allowlist, k) {
					px := anthropic.ToolInputSchemaParam{}
					_ = copier.Copy(&px, v.Schema)
					tx = append(tx, anthropic.ToolUnionParam{
						OfTool: &anthropic.ToolParam{
							Name:        k,
							Description: anthropic.Opt(v.Description),
							InputSchema: px,
						},
					})
				}
			}
		case frags.ToolTypeInternetSearch:
			tx = append(tx, anthropic.ToolUnionParam{
				OfWebSearchTool20260209: &anthropic.WebSearchTool20260209Param{
					MaxUses: anthropic.Int(2),
				},
			})
		}
	}
	return tx, nil
}

func (d *Ai) SetFunctions(functions frags.ExternalFunctions) {
	d.Functions = functions
}

func (d *Ai) RunFunction(ctx *util.FragsContext, functionCall frags.FunctionCaller, runner frags.ExportableRunner) (any, error) {
	return runner.RunFunction(ctx, functionCall.Name, functionCall.Args)
}
