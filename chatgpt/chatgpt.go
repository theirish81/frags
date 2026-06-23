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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go/v5"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/schema"
	"github.com/theirish81/frags/util"
)

const defaultModel = "gpt-5"
const engine = "chatgpt"

type Ai struct {
	apiKey       string
	baseURL      string
	httpClient   *HttpClient
	systemPrompt string
	config       Config
	content      Messages
	Functions    frags.ExternalFunctions
	files        map[string]string
	uploadMutex  *sync.Mutex
}
type Config struct {
	Model         string        `yaml:"model" json:"model"`
	Attempts      int           `yaml:"attempts" json:"attempts"`
	RetryDelay    time.Duration `yaml:"retryDelay" json:"retryDelay"`
	ThinkingLevel *string       `yaml:"thinkingLevel" json:"thinkingLevel"`
}

func DefaultConfig() Config {
	return Config{
		Model:      defaultModel,
		Attempts:   3,
		RetryDelay: 2 * time.Second,
	}
}

func (d *Ai) SetSystemPrompt(prompt string) {
	d.systemPrompt = prompt
}

func NewAI(baseURL string, apiKey string, config Config) *Ai {
	return &Ai{
		apiKey:      apiKey,
		baseURL:     baseURL,
		config:      config,
		content:     make([]Message, 0),
		Functions:   frags.ExternalFunctions{},
		files:       make(map[string]string),
		httpClient:  NewHttpClient(baseURL, apiKey),
		uploadMutex: &sync.Mutex{},
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
		uploadMutex:  d.uploadMutex,
	}
}

func (d *Ai) Ask(ctx *util.FragsContext, text string, sx *schema.Schema, tools frags.ToolDefinitions,
	runner frags.ExportableRunner, rx ...resources.ResourceData) ([]byte, error) {

	chatGptTools, err := d.configureTools(tools)
	if err != nil {
		return nil, err
	}
	runner.Logger().Debug(log.NewEvent(log.GenericEventType, log.AiComponent).WithEngine(engine).WithMessage("configured tools").WithContent(tools))
	msg := NewUserMessage(text)
	for _, r := range rx {
		switch r.MediaType {
		case util.MediaText:
			// For some unexplainable reasons, the Upload API doesn't like text files, so the best thing we can do is
			// attach them to the message itself.
			msg.Content.InsertTextPart(fmt.Sprintf("=== %s === \n%s\n ===\n", r.Identifier, string(r.ByteContent)))
		default:
			if err = func() error {
				d.uploadMutex.Lock()
				defer d.uploadMutex.Unlock()
				fid, ok := d.files[r.Identifier]
				if !ok {
					fd, err := d.httpClient.FileUpload(ctx, r.Identifier, r.ByteContent)
					if err != nil {
						return err
					}
					fid = fd.Id
					d.files[r.Identifier] = fd.Id
				}
				msg.Content.InsertFileMessage(fid)
				return nil
			}(); err != nil {
				return nil, err
			}

		}
	}
	d.content = append(d.content, msg)
	keepGoing := true
	out := ""
	counter := 0
	for keepGoing {
		counter++
		if counter >= 10 {
			return nil, errors.New("loop detected. Too many iterations")
		}

		runner.Logger().Debug(log.NewEvent(log.StartEventType, log.AiComponent).WithMessage("generating content").WithContent(d.content[len(d.content)-1]).WithEngine(engine))
		req := NewResponseRequest(d.config.Model, d.content, d.systemPrompt, chatGptTools, sx)
		if d.config.ThinkingLevel != nil {
			req.Reasoning = &ReasoningConfig{
				Effort: *d.config.ThinkingLevel,
			}
		}

		var response Response
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
				runner.Logger().Info(log.NewEvent(log.GenericEventType, log.AiComponent).WithMessage("ChatGPT infrastructure is overloaded, retrying...").WithEngine(engine).WithErr(err).WithIteration(int(attempt)))
			})).Do(func() error {
			response, err = d.httpClient.PostResponses(ctx, req)
			return err
		}); err != nil {
			return nil, err
		}

		for _, item := range response.Output {
			d.content = append(d.content, item)
		}
		if response.HasFunctionCalls() {
			if err := d.handleFunctionCalls(ctx, response, runner); err != nil {
				return nil, err
			}
		} else {
			content := response.Output.Last().Content
			runner.Logger().Debug(log.NewEvent(log.EndEventType, log.AiComponent).WithMessage("generated content").WithContent(content).WithEngine(engine))
			out = content.First().Text
			keepGoing = false
			if strings.Contains(out, "[FATAL]") {
				err = errors.New(out)
			}
		}

	}
	return []byte(out), err
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

func (d *Ai) SetFunctions(functions frags.ExternalFunctions) {
	d.Functions = functions
}

type FunctionResponseMap map[string]any

func NewFunctionResponseMap(data any, err error) FunctionResponseMap {
	res := FunctionResponseMap{
		"output": data,
	}
	if err != nil {
		res["error"] = err.Error()
	}
	res["system_guidance"] = `
* in case of success, ignore this guidance
* if recoverable failure, retry function calling MAX 3 times per function, adjusting the parameters as necessary.
* if the error is deemed not solvable, not recoverable or we hit the 3rd attempt, respond with the failure reasons, preceded by the flag [FATAL]`
	return res
}

func (d *Ai) handleFunctionCalls(ctx *util.FragsContext, responseMessage Response, runner frags.ExportableRunner) error {
	for _, fc := range responseMessage.FunctionCalls() {
		res, err := d.RunFunction(ctx, frags.FunctionCaller{Name: fc.Name, Args: fc.Arguments.GetMap()}, runner)
		data, _ := json.Marshal(NewFunctionResponseMap(res, err))

		d.content = append(d.content, Message{
			Type:   PartTypeFunctionCallOutput,
			CallID: fc.CallID,
			Output: string(data),
		})
	}
	return nil
}

func (d *Ai) RunFunction(ctx *util.FragsContext, functionCall frags.FunctionCaller, runner frags.ExportableRunner) (any, error) {
	return runner.RunFunction(ctx, functionCall.Name, functionCall.Args)
}
