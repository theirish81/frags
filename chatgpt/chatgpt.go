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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/theirish81/frags"
)

const defaultModel = "gpt-5"

type Ai struct {
	apiKey       string
	baseURL      string
	httpClient   *HttpClient
	systemPrompt string
	config       Config
	content      Messages
	Functions    frags.Functions
	log          *slog.Logger
	files        map[string]string
}
type Config struct {
	Model string `yaml:"model" json:"model"`
}

func DefaultConfig() Config {
	return Config{
		Model: defaultModel,
	}
}

func (d *Ai) SetSystemPrompt(prompt string) {
	d.systemPrompt = prompt
}

func NewAI(baseURL string, apiKey string, config Config, log *slog.Logger) *Ai {
	return &Ai{
		apiKey:     apiKey,
		baseURL:    baseURL,
		config:     config,
		content:    make([]Message, 0),
		Functions:  frags.Functions{},
		files:      make(map[string]string),
		log:        log,
		httpClient: NewHttpClient(baseURL, apiKey),
	}
}

// New creates a new Ai instance and returns it
func (d *Ai) New() frags.Ai {
	return &Ai{
		httpClient:   d.httpClient,
		baseURL:      d.baseURL,
		apiKey:       d.apiKey,
		content:      make([]Message, 0),
		Functions:    d.Functions,
		config:       d.config,
		systemPrompt: d.systemPrompt,
		files:        d.files,
		log:          d.log,
	}
}

func (d *Ai) Ask(ctx context.Context, text string, schema *frags.Schema, tools frags.ToolDefinitions,
	runner frags.ExportableRunner, resources ...frags.ResourceData) ([]byte, error) {

	chatGptTools, err := d.configureTools(tools)
	if err != nil {
		return nil, err
	}
	msg := NewUserMessage(text)
	for _, r := range resources {
		switch r.MediaType {
		case frags.MediaText:
			// For some unexplainable reasons, the Upload API doesn't like text files, so the best thing we can do is
			// attach them to the message itself.
			msg.Content.InsertTextPart(fmt.Sprintf("=== %s === \n%s ===\n", r.Identifier, string(r.ByteContent)))
		default:
			fid, ok := d.files[r.Identifier]
			if !ok {
				fd, err := d.httpClient.FileUpload(ctx, r.Identifier, r.ByteContent)
				if err != nil {
					return nil, err
				}
				fid = fd.Id
				d.files[r.Identifier] = fd.Id
			}
			msg.Content.InsertFileMessage(fid)
		}
	}
	d.content = append(d.content, msg)
	keepGoing := true
	out := ""
	for keepGoing {
		req := NewResponseRequest(d.config.Model, d.content, d.systemPrompt, chatGptTools, schema)

		response, err := d.httpClient.PostResponses(ctx, req)
		if err != nil {
			return nil, err
		}
		d.content = append(d.content, response.Output.Last())
		if response.HasFunctionCalls() {
			if err := d.handleFunctionCalls(response, runner); err != nil {
				return nil, err
			}
		} else {
			content := response.Output.Last().Content
			out = content.First().Text
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
					Type:        PartTypeFunction,
					Description: description,
					Parameters:  pSchema,
				})
			}
		case frags.ToolTypeMCP, frags.ToolTypeCollection:
			for k, v := range d.Functions.ListByCollection(tool.Name) {
				if tool.Allowlist == nil || slices.Contains(*tool.Allowlist, k) {
					oaTools = append(oaTools, ChatGptTool{
						Type:        PartTypeFunction,
						Name:        k,
						Description: v.Description,
						Parameters:  v.Schema,
					})
				}
			}
		case frags.ToolTypeInternetSearch:
			oaTools = append(oaTools, ChatGptTool{
				Type: "web_search",
			})
		}

	}
	return oaTools, nil
}

func (d *Ai) SetFunctions(functions frags.Functions) {
	d.Functions = functions
}

func (d *Ai) handleFunctionCalls(responseMessage Response, runner frags.ExportableRunner) error {
	for _, fc := range responseMessage.FunctionCalls() {
		res, err := d.RunFunction(frags.FunctionCall{Name: fc.Name, Args: fc.Arguments.GetMap()}, runner)
		if err != nil {
			return err
		}
		data, err := json.Marshal(res)
		if err != nil {
			return err
		}
		d.content = append(d.content, Message{
			Type:   PartTypeFunctionCallOutput,
			CallID: fc.CallID,
			Output: string(data),
		})
	}
	return nil
}

func (d *Ai) RunFunction(functionCall frags.FunctionCall, runner frags.ExportableRunner) (any, error) {
	if fx, ok := d.Functions[functionCall.Name]; ok {
		functionSignature := fmt.Sprintf("%s(%v)", functionCall.Name, functionCall.Args)
		d.log.Debug("invoking function", "ai", "chatgpt", "function", functionSignature)
		res, err := fx.Run(functionCall.Args, runner)
		d.log.Debug("function result", "ai", "chatgpt", "function", functionSignature, "result", res, "error", err)
		return res, err
	}
	return nil, errors.New("function not found")
}
