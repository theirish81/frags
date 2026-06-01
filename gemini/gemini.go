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

package gemini

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/avast/retry-go/v5"
	"github.com/jinzhu/copier"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/schema"
	"github.com/theirish81/frags/util"
	"google.golang.org/api/googleapi"
	"google.golang.org/genai"
)

const engine = "gemini"

// Gemini defaults
const temperature float32 = 0.1
const topK float32 = 40
const topP float32 = 0.9

const jsonContentType = "application/json"
const textContentType = "text/plain"
const defaultModel = "gemini-2.5-flash"

// Ai is a wrapper around the genai client for Frags
type Ai struct {
	client       *genai.Client
	systemPrompt string
	content      []*genai.Content
	Functions    frags.ExternalFunctions
	config       Config
}

type Config struct {
	Model         string               `yaml:"model" json:"model"`
	Temperature   float32              `yaml:"temperature" json:"temperature"`
	TopK          float32              `yaml:"topK" json:"topK"`
	TopP          float32              `yaml:"topP" json:"topP"`
	Attempts      int                  `yaml:"attempts" json:"attempts"`
	RetryDelay    time.Duration        `yaml:"retryDelay" json:"retryDelay"`
	ThinkingLevel *genai.ThinkingLevel `yaml:"thinkingLevel" json:"thinkingLevel"`
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
func NewAI(client *genai.Client, config Config) *Ai {
	return &Ai{
		client:    client,
		content:   make([]*genai.Content, 0),
		Functions: frags.ExternalFunctions{},
		config:    config,
	}
}

// New creates a new Ai instance and returns it
func (d *Ai) New() frags.Ai {
	return &Ai{
		client:       d.client,
		content:      make([]*genai.Content, 0),
		Functions:    d.Functions,
		config:       d.config,
		systemPrompt: d.systemPrompt,
	}
}

// Ask performs a query against the Gemini API, according to the Frags interface
func (d *Ai) Ask(ctx *util.FragsContext, text string, sx *schema.Schema, tools frags.ToolDefinitions,
	runner frags.ExportableRunner, rx ...resources.ResourceData) ([]byte, error) {
	if len(text) == 0 {
		if sx == nil && len(d.content) > 0 {
			return []byte(joinParts(d.content[len(d.content)-1].Parts)), nil
		}
		return nil, fmt.Errorf("no prompt provided but was required")
	}
	parts := make([]*genai.Part, 0)
	for _, resource := range rx {
		runner.Logger().Debug(log.NewEvent(log.LoadEventType, log.AiComponent).WithResource(resource.Identifier).WithEngine(engine))
		content := resource.ByteContent
		if resource.MediaType == util.MediaText {
			content = []byte(fmt.Sprintf("<Document name=\"%s\"><![CDATA[ %s ]]></Document>\n", resource.Identifier, string(resource.ByteContent)))
		}
		parts = append(parts, genai.NewPartFromBytes(content, resource.MediaType))
	}
	parts = append(parts, genai.NewPartFromText(text))
	genAiSchema := &genai.Schema{}
	ct := jsonContentType
	if sx == nil {
		ct = textContentType
		genAiSchema = nil
	} else {
		if err := sx.CopyTo(genAiSchema); err != nil {
			return nil, err
		}
	}
	newMsg := genai.NewContentFromParts(parts, genai.RoleUser)

	tx, err := d.configureTools(tools)
	runner.Logger().Debug(log.NewEvent(log.GenericEventType, log.AiComponent).WithEngine(engine).WithMessage("configured tools").WithContent(tools))
	if err != nil {
		return nil, err
	}
	cfg := genai.GenerateContentConfig{
		ResponseMIMEType: ct,
		ResponseSchema:   genAiSchema,
		Temperature:      &d.config.Temperature,
		TopK:             &d.config.TopK,
		TopP:             &d.config.TopP,
		Tools:            tx,
		SafetySettings: []*genai.SafetySetting{
			{
				Category:  genai.HarmCategoryDangerousContent,
				Threshold: genai.HarmBlockThresholdBlockNone,
			},
		},
	}
	if len(d.systemPrompt) > 0 {
		cfg.SystemInstruction = genai.NewContentFromText(d.systemPrompt, "system")
	}
	if d.config.ThinkingLevel != nil {
		cfg.ThinkingConfig = &genai.ThinkingConfig{
			ThinkingLevel:   *d.config.ThinkingLevel,
			IncludeThoughts: false,
		}
	}
	keepGoing := true
	out := ""
	counter := 0
	d.content = append(d.content, newMsg)
	for keepGoing {
		counter++
		if counter >= 10 {
			return nil, errors.New("loop detected. Too many iterations")
		}
		runner.Logger().Debug(log.NewEvent(log.StartEventType, log.AiComponent).WithMessage("generating content").WithContent(joinParts(d.content[len(d.content)-1].Parts)).WithEngine(engine))
		var res *genai.GenerateContentResponse
		if d.config.Attempts <= 0 {
			d.config.Attempts = 1
		}
		if err = retry.New(retry.Attempts(uint(d.config.Attempts)), retry.Delay(d.config.RetryDelay), retry.Context(ctx),
			retry.DelayType(retry.BackOffDelay), retry.RetryIf(func(err error) bool {
				if strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
					return true
				}
				// 1. Handle HTTP 429 and 5xx
				var gerr *googleapi.Error
				if errors.As(err, &gerr) {
					return gerr.Code == http.StatusTooManyRequests || gerr.Code >= 500
				}
				// 2. Handle Network/Connectivity issues
				var netErr net.Error
				if errors.As(err, &netErr) {
					return true
				}
				return false
			}), retry.OnRetry(func(attempt uint, err error) {
				runner.Logger().Info(log.NewEvent(log.GenericEventType, log.AiComponent).WithMessage("Google Gemini infrastructure is overloaded, retrying...").WithEngine(engine).WithErr(err).WithIteration(int(attempt)))
			})).Do(func() error {
			res, err = d.client.Models.GenerateContent(ctx, d.config.Model, d.content, &cfg)
			return err
		}); err != nil {
			return nil, err
		}
		d.content = append(d.content, res.Candidates[0].Content)
		if res.FunctionCalls() != nil && len(res.FunctionCalls()) > 0 {
			// It seems that if function calls are more than one, Gemini expects all the responses in one content,
			// with multiple parts, one per function response.
			userTurn := &genai.Content{
				Role:  genai.RoleUser,
				Parts: []*genai.Part{},
			}
			for _, fc := range res.FunctionCalls() {
				fres, ferr := d.RunFunction(ctx, frags.FunctionCaller{Name: fc.Name, Args: fc.Args}, runner)
				// adding a function response to the user turn object
				userTurn.Parts = append(userTurn.Parts, &genai.Part{
					FunctionResponse: &genai.FunctionResponse{
						ID:       fc.ID,
						Name:     fc.Name,
						Response: NewFunctionResponseMap(fres, ferr),
					},
				})
			}
			d.content = append(d.content, userTurn)
			keepGoing = true
		} else {
			keepGoing = false
			candidate := res.Candidates[0]
			out = joinParts(candidate.Content.Parts)
			internetSearch := candidate.GroundingMetadata != nil
			runner.Logger().Debug(log.NewEvent(log.EndEventType, log.AiComponent).WithMessage("generated content").WithEngine(engine).WithContent(out).WithArg("internet_search", internetSearch))
			if strings.Contains(out, "[FATAL]") {
				err = errors.New(out)
			}
		}
	}
	return []byte(out), err
}

func (d *Ai) configureTools(tools frags.ToolDefinitions) ([]*genai.Tool, error) {
	tx := make([]*genai.Tool, 0)
	fd := make([]*genai.FunctionDeclaration, 0)
	for _, tool := range tools {
		switch tool.Type {
		case frags.ToolTypeFunction:
			if fx, found := d.Functions[tool.Name]; found {
				pSchema := fx.Schema
				if tool.InputSchema != nil {
					pSchema = tool.InputSchema
				}
				genAiPSchema := &genai.Schema{}
				if err := pSchema.CopyTo(genAiPSchema); err != nil {
					return nil, err
				}
				description := fx.Description
				if len(tool.Description) > 0 {
					description = tool.Description
				}
				fd = append(fd, &genai.FunctionDeclaration{
					Name:        tool.Name,
					Description: description,
					Parameters:  genAiPSchema,
				})
			}
		default:
			for k, v := range d.Functions.ListByCollection(tool.Name) {
				var genAiInputSchema *genai.Schema
				if v.Schema != nil {
					genAiInputSchema = &genai.Schema{}

					if err := v.Schema.CopyTo(genAiInputSchema); err != nil {
						return nil, err
					}
				}
				var genAiOutputSchema *genai.Schema
				if v.OutputSchema != nil {
					genAiOutputSchema = &genai.Schema{}

					_ = v.OutputSchema.CopyTo(genAiOutputSchema)
				}
				if tool.Allowlist == nil || slices.Contains(*tool.Allowlist, k) {
					fd = append(fd, &genai.FunctionDeclaration{
						Name:        k,
						Description: v.Description,
						Parameters:  genAiInputSchema,
						Response:    genAiOutputSchema,
					})
				}

			}
		case frags.ToolTypeInternetSearch:
			tx = append(tx, &genai.Tool{
				GoogleSearch: &genai.GoogleSearch{},
			})
		}
	}
	if len(fd) > 0 {
		tx = append(tx, &genai.Tool{
			FunctionDeclarations: fd,
		})
	}
	return tx, nil
}

func (d *Ai) SetFunctions(functions frags.ExternalFunctions) {
	d.Functions = functions
}

func joinParts(parts []*genai.Part) string {
	out := ""
	for _, part := range parts {
		out += part.Text
	}
	return out
}

func (d *Ai) RunFunction(ctx *util.FragsContext, functionCall frags.FunctionCaller, runner frags.ExportableRunner) (any, error) {
	return runner.RunFunction(ctx, functionCall.Name, functionCall.Args)
}

func (d *Ai) SchemaConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		{
			SrcType: []any{},
			DstType: []string{},
			Fn: func(src any) (any, error) {
				s := src.([]any)
				res := make([]string, len(s))
				for i, v := range s {
					res[i] = fmt.Sprint(v)
				}
				return res, nil
			},
		},
	}
}
