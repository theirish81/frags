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

package chatgpt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/theirish81/frags"
)

const temperature float32 = 0.1
const topK float32 = 40
const topP float32 = 0.9

const jsonContentType = "application/json"
const textContentType = "text/plain"
const defaultModel = openai.ChatModelGPT5

type Ai struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	systemPrompt string
	config       Config
	content      []InputItem
	Functions    frags.Functions
	log          *slog.Logger
}
type Config struct {
	Model       string  `yaml:"model" json:"model"`
	Temperature float32 `yaml:"temperature" json:"temperature"`
	TopK        float32 `yaml:"topK" json:"top_k"`
	TopP        float32 `yaml:"topP" json:"top_p"`
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

func NewAI(baseURL string, apiKey string, config Config, log *slog.Logger) *Ai {
	return &Ai{
		apiKey:    apiKey,
		baseURL:   baseURL,
		config:    config,
		content:   make([]InputItem, 0),
		Functions: frags.Functions{},
		log:       log,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// New creates a new Ai instance and returns it
func (d *Ai) New() frags.Ai {
	return &Ai{
		httpClient:   d.httpClient,
		baseURL:      d.baseURL,
		apiKey:       d.apiKey,
		content:      make([]InputItem, 0),
		Functions:    d.Functions,
		config:       d.config,
		systemPrompt: d.systemPrompt,
		log:          d.log,
	}
}

func (d *Ai) Ask(ctx context.Context, text string, schema *frags.Schema, tools frags.ToolDefinitions,
	runner frags.ExportableRunner, resources ...frags.ResourceData) ([]byte, error) {
	chatGptTools, err := d.configureTools(tools)
	if err != nil {
		return nil, err
	}
	d.content = append(d.content, InputItem{
		Role: "user",
		Content: []ContentPart{
			{
				Type: "input_text",
				Text: text,
			},
		},
	})
	keepGoing := true
	out := ""
	for keepGoing {
		req := ResponseRequest{
			Model: d.config.Model,
			Input: d.content,
			Tools: chatGptTools,
		}
		if schema != nil {
			req.Text = &Text{
				Format: &ResponseFormat{
					Name:   "response",
					Type:   "json_schema",
					Schema: schema,
				},
			}
		}
		jsonData, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request: %w", err)
		}

		httpReq, err := http.NewRequest("POST", d.baseURL+"/responses", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("error creating HTTP request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)

		resp, err := d.httpClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("error sending request: %w", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		var response Response
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("error parsing response: %w", err)
		}
		item := response.Output[len(response.Output)-1]
		d.content = append(d.content, item)
		if response.HasFunctionCalls() {
			if err := d.handleFunctionCalls(response, runner); err != nil {
				return nil, err
			}
		} else {
			out = item.Content[0].Text
			keepGoing = false
		}

	}
	return []byte(out), nil
}

func (d *Ai) configureTools(tools frags.ToolDefinitions) ([]ChatGptTool, error) {
	oaTools := make([]ChatGptTool, 0)
	for _, tool := range tools {
		switch tool.Type {
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
				oaTools = append(oaTools, ChatGptTool{
					Name:        tool.Name,
					Type:        "function",
					Description: description,
					Parameters:  pSchema,
				})
			}
		case frags.ToolTypeMCP, frags.ToolTypeCollection:
			for k, v := range d.Functions.ListByCollection(tool.Name) {
				if tool.Allowlist == nil || slices.Contains(*tool.Allowlist, k) {
					oaTools = append(oaTools, ChatGptTool{
						Type:        "function",
						Name:        k,
						Description: v.Description,
						Parameters:  v.Schema,
					})
				}
			}
			/*case frags.ToolTypeInternetSearch:
			tx = append(tx, &genai.Tool{
				GoogleSearch: &genai.GoogleSearch{},
			})

			*/
		}

	}
	/*if len(fd) > 0 {
		tx = append(tx, &genai.Tool{
			FunctionDeclarations: fd,
		})
	}
	return tx, nil
	*/
	return oaTools, nil
}

func (d *Ai) SetFunctions(functions frags.Functions) {
	d.Functions = functions
}

func (d *Ai) handleFunctionCalls(responseMessage Response, runner frags.ExportableRunner) error {
	for _, fc := range responseMessage.FunctionCalls() {
		res, err := d.RunFunction(frags.FunctionCall{Name: fc.Name, Args: fc.Arguments.Map}, runner)
		if err != nil {
			return err
		}
		data, err := json.Marshal(res)
		if err != nil {
			return err
		}
		d.content = append(d.content, InputItem{
			Type:   "function_call_output",
			CallID: fc.CallID,
			Output: string(data),
		})
	}
	return nil
}

func (d *Ai) RunFunction(functionCall frags.FunctionCall, runner frags.ExportableRunner) (map[string]any, error) {
	if fx, ok := d.Functions[functionCall.Name]; ok {
		functionSignature := fmt.Sprintf("%s(%v)", functionCall.Name, functionCall.Args)
		d.log.Debug("invoking function", "ai", "gemini", "function", functionSignature)
		res, err := fx.Run(functionCall.Args, runner)
		d.log.Debug("function result", "ai", "gemini", "function", functionSignature, "result", res, "error", err)
		return res, err
	}
	return nil, errors.New("function not found")
}
