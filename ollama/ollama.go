package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/theirish81/frags"
)

const defaultModel = "qwen3:latest"
const temperature float32 = 0.1
const topK float32 = 40
const topP float32 = 0.9

// Ai implements the Ai interface for Ollama.
type Ai struct {
	client       *http.Client
	baseURL      string
	config       Config
	messages     []Message
	systemPrompt string
	Functions    frags.Functions
	log          *slog.Logger
}

type Config struct {
	Model       string  `yaml:"model" json:"model"`
	Temperature float32 `yaml:"temperature" json:"temperature"`
	TopK        float32 `yaml:"topK" json:"top_k"`
	TopP        float32 `yaml:"topP" json:"top_p"`
	NumPredict  int     `yaml:"numPredict" json:"num_predict"`
}

func DefaultConfig() Config {
	return Config{
		Model:       defaultModel,
		Temperature: temperature,
		TopK:        topK,
		TopP:        topP,
		NumPredict:  1024,
	}
}

// NewAI creates a new Ai instance
func NewAI(baseURL string, config Config, log *slog.Logger) *Ai {
	return &Ai{
		baseURL: baseURL,
		config:  config,
		client: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 10,
			},
		},
		Functions: frags.Functions{},
		messages:  make([]Message, 0),
		log:       log,
	}
}

// New creates a new Ai instance and returns it
func (d *Ai) New() frags.Ai {
	return &Ai{
		client:       d.client,
		baseURL:      d.baseURL,
		config:       d.config,
		messages:     make([]Message, 0),
		Functions:    d.Functions,
		systemPrompt: d.systemPrompt,
		log:          d.log,
	}
}

// Ask performs a query against the Ollama API, according to the Frags interface
func (d *Ai) Ask(ctx context.Context, text string, schema *frags.Schema, tools frags.ToolDefinitions,
	runner frags.ExportableRunner, resources ...frags.ResourceData) ([]byte, error) {
	if len(d.systemPrompt) > 0 && len(d.messages) == 0 {
		d.messages = append(d.messages, Message{
			Content: d.systemPrompt,
			Role:    "system",
		})
	}
	message := Message{
		Content: "",
		Role:    "user",
	}
	for _, r := range resources {
		if r.MediaType != frags.MediaText {
			return nil, errors.New("ollama only supports text resources")
		}
		message.Content += message.Content + " === " + r.Identifier + " === \n" + string(r.ByteContent) + "\n===\n"
		d.log.Debug("adding file resource", "ai", "ollama", "resource", r.Identifier)
	}
	message.Content += "\n" + text
	d.messages = append(d.messages, message)
	request := Request{
		Messages: d.messages,
		Model:    d.config.Model,
		Think:    false,
		Format:   schema,
		Tools:    make([]ToolDefinition, 0),
		Options: Options{
			NumPredict:  d.config.NumPredict,
			Temperature: d.config.Temperature,
			TopK:        d.config.TopK,
			TopP:        d.config.TopP,
		},
	}
	request.Tools, _ = d.configureTools(tools)
	d.log.Debug("configured tools", "ai", "ollama", "tools", request.Tools)
	keepGoing := true
	out := ""
	for keepGoing {
		request.Messages = d.messages
		err := func() error {
			d.log.Debug("generating content", "ai", "ollama", "message", request.Messages[len(request.Messages)-1])
			responseMessage, err := d.sendRequest(ctx, request)
			if err != nil {
				return err
			}
			d.messages = append(d.messages, responseMessage.Message)
			if len(responseMessage.Message.ToolCalls) > 0 {
				err := d.handleFunctionCall(responseMessage, runner)
				if err != nil {
					return err
				}
			} else {
				out = responseMessage.Message.Content
				d.log.Debug("generated response", "ai", "ollama", "response", out)
				keepGoing = false
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return []byte(out), nil
}

func (d *Ai) handleFunctionCall(responseMessage Response, runner frags.ExportableRunner) error {
	for _, fc := range responseMessage.Message.ToolCalls {
		res, err := d.RunFunction(frags.FunctionCall{Name: fc.Function.Name, Args: fc.Function.Arguments}, runner)
		if err != nil {
			return err
		}
		content, err := json.Marshal(res)
		if err != nil {
			return err
		}
		d.messages = append(d.messages, Message{
			Role:       "tool",
			Content:    string(content),
			ToolCallID: fc.ID,
		})
	}
	return nil
}

func (d *Ai) SetSystemPrompt(systemPrompt string) {
	d.systemPrompt = systemPrompt
}

func (d *Ai) SetFunctions(functions frags.Functions) {
	d.Functions = functions
}

func (d *Ai) sendRequest(ctx context.Context, request Request) (Response, error) {
	requestBody, err := json.Marshal(request)
	response := Response{}
	if err != nil {
		return response, err
	}
	reader := bytes.NewReader(requestBody)
	apiUrl, err := url.Parse(d.baseURL + "/api/chat")
	if err != nil {
		return response, err
	}
	req := http.Request{
		Method: "POST",
		URL:    apiUrl,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(reader),
	}

	res, err := d.client.Do(req.WithContext(ctx))
	if err != nil {
		return response, err
	}
	defer func() {
		if res.Body != nil {
			_ = res.Body.Close()
		}
	}()
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(responseBody, &response)
	return response, err
}

func (d *Ai) configureTools(tools frags.ToolDefinitions) ([]ToolDefinition, error) {
	tx := make([]ToolDefinition, 0)
	for _, tool := range tools {
		switch tool.Type {
		case frags.ToolTypeMCP, frags.ToolTypeCollection:
			for k, v := range d.Functions.ListByCollection(tool.Name) {
				if tool.Allowlist == nil || slices.Contains(*tool.Allowlist, k) {
					tx = append(tx, ToolDefinition{
						Type: "function",
						Function: FunctionDef{
							Name:        k,
							Description: v.Description,
							Parameters:  v.Schema,
						},
					})
				}
			}
		case frags.ToolTypeFunction:
			if fx, found := d.Functions[tool.Name]; found {
				pSchema := fx.Schema
				if tool.InputSchema != nil {
					pSchema = tool.InputSchema
				}
				description := fx.Description
				if len(tool.Description) > 0 {
					description = tool.Description
				}
				tx = append(tx, ToolDefinition{
					Type: "function",
					Function: FunctionDef{
						Name:        tool.Name,
						Description: description,
						Parameters:  pSchema,
					},
				})
			}
		}
	}
	return tx, nil
}

func (d *Ai) RunFunction(functionCall frags.FunctionCall, runner frags.ExportableRunner) (any, error) {
	if fx, ok := d.Functions[functionCall.Name]; ok {
		d.log.Debug("invoking function", "ai", "ollama", "function", fmt.Sprintf("%s(%v)", functionCall.Name, functionCall.Args))
		res, err := fx.Run(functionCall.Args, runner)
		d.log.Debug("function result", "ai", "ollama", "function", fmt.Sprintf("%s(%v)", functionCall.Name, functionCall.Args), "result", res, "error", err)
		return res, err
	}
	return nil, errors.New("function not found")

}
