package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/theirish81/frags"
)

const defaultModel = "qwen3:latest"
const temperature float32 = 0.1
const topK float32 = 40
const topP float32 = 0.9

// Ai implements the Ai interface for Ollama.
type Ai struct {
	client    *http.Client
	baseURL   string
	config    Config
	messages  []Message
	Functions frags.Functions
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
func NewAI(baseURL string, config Config) *Ai {
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
	}
}

// New creates a new Ai instance and returns it
func (d *Ai) New() frags.Ai {
	return &Ai{
		client:    d.client,
		baseURL:   d.baseURL,
		config:    d.config,
		messages:  make([]Message, 0),
		Functions: d.Functions,
	}
}

// Ask performs a query against the Ollama API, according to the Frags interface
func (d *Ai) Ask(ctx context.Context, text string, schema *frags.Schema, tools frags.Tools, resources ...frags.ResourceData) ([]byte, error) {
	message := Message{
		Content: "",
		Role:    "user",
	}
	for _, r := range resources {
		if r.MediaType != "text/plain" {
			return nil, errors.New("ollama only supports text resources")
		}
		message.Content += message.Content + " === " + r.Identifier + " === \n" + string(r.Data) + "\n"
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
	keepGoing := true
	out := ""
	for keepGoing {
		request.Messages = d.messages
		err := func() error {
			responseMessage, err := d.sendRequest(ctx, request)
			if err != nil {
				return err
			}
			d.messages = append(d.messages, responseMessage.Message)
			if len(responseMessage.Message.ToolCalls) > 0 {
				err := d.handleFunctionCall(responseMessage)
				if err != nil {
					return err
				}
			} else {
				out = responseMessage.Message.Content
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

func (d *Ai) handleFunctionCall(responseMessage Response) error {
	for _, fc := range responseMessage.Message.ToolCalls {
		res, err := d.Functions[fc.Function.Name].Run(fc.Function.Arguments)
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

func (d *Ai) configureTools(tools frags.Tools) ([]ToolDefinition, error) {
	tx := make([]ToolDefinition, 0)
	for _, tool := range tools {
		switch tool.Type {
		case frags.ToolTypeMCP:
			for k, v := range d.Functions.ListByServer(tool.ServerName) {
				tx = append(tx, ToolDefinition{
					Type: "function",
					Function: FunctionDef{
						Name:        k,
						Description: v.Description,
						Parameters:  v.Schema,
					},
				})
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
