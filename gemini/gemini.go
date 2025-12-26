package gemini

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/jinzhu/copier"
	"github.com/theirish81/frags"
	"google.golang.org/genai"
)

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
	Functions    frags.Functions
	config       Config
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

// NewAI creates a new Ai wrapper
func NewAI(client *genai.Client, config Config, log *slog.Logger) *Ai {
	return &Ai{
		client:    client,
		content:   make([]*genai.Content, 0),
		Functions: frags.Functions{},
		config:    config,
		log:       log,
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
		log:          d.log,
	}
}

// Ask performs a query against the Gemini API, according to the Frags interface
func (d *Ai) Ask(ctx context.Context, text string, schema *frags.Schema, tools frags.Tools,
	runner frags.ExportableRunner, resources ...frags.ResourceData) ([]byte, error) {
	parts := make([]*genai.Part, 0)
	for _, resource := range resources {
		d.log.Debug("adding file resource", "ai", "gemini", "resource", resource.Identifier)
		parts = append(parts, genai.NewPartFromBytes(resource.Data, resource.MediaType))
	}
	parts = append(parts, genai.NewPartFromText(text))
	genAiSchema := &genai.Schema{}
	ct := jsonContentType
	if schema == nil {
		ct = textContentType
		genAiSchema = nil
	} else {
		if err := copier.Copy(genAiSchema, schema); err != nil {
			return nil, err
		}
	}
	newMsg := genai.NewContentFromParts(parts, genai.RoleUser)

	tx, err := d.configureTools(tools)
	d.log.Debug("configured tools", "ai", "gemini", "tools", tools)
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
	}
	if len(d.systemPrompt) > 0 {
		cfg.SystemInstruction = genai.NewContentFromText(d.systemPrompt, "system")
	}
	keepGoing := true
	out := ""
	d.content = append(d.content, newMsg)
	for keepGoing {
		d.log.Debug("generating content", "ai", "gemini", "message", PartToLoggableText(d.content[len(d.content)-1]))
		res, err := d.client.Models.GenerateContent(ctx, d.config.Model, d.content, &cfg)
		if err != nil {
			return nil, err
		}
		d.content = append(d.content, res.Candidates[0].Content)
		if res.FunctionCalls() != nil {
			for _, fc := range res.FunctionCalls() {
				d.content = append(d.content, genai.NewContentFromFunctionCall(fc.Name, fc.Args, genai.RoleModel))

				fres, ferr := d.RunFunction(frags.FunctionCall{Name: fc.Name, Args: fc.Args}, runner)
				if ferr != nil {
					return nil, ferr
				} else {
					d.content = append(d.content, genai.NewContentFromFunctionResponse(fc.Name, fres, genai.RoleUser))
				}
			}
			keepGoing = true
		} else {
			keepGoing = false
			candidate := res.Candidates[0]
			d.content = append(d.content, candidate.Content)
			out = joinParts(candidate.Content.Parts)
			internetSearch := candidate.GroundingMetadata != nil
			d.log.Debug("generated response", "ai", "gemini", "response", out, "internet_search", internetSearch)
		}
	}
	return []byte(out), nil
}

func (d *Ai) configureTools(tools frags.Tools) ([]*genai.Tool, error) {
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
				if err := copier.Copy(genAiPSchema, pSchema); err != nil {
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
		case frags.ToolTypeMCP, frags.ToolTypeCollection:
			for k, v := range d.Functions.ListByCollection(tool.Name) {
				var genAiPSchema *genai.Schema
				if v.Schema != nil {
					genAiPSchema = &genai.Schema{}
					if err := copier.Copy(genAiPSchema, v.Schema); err != nil {
						return nil, err
					}
				}
				if tool.Allowlist == nil || slices.Contains(*tool.Allowlist, k) {
					fd = append(fd, &genai.FunctionDeclaration{
						Name:        k,
						Description: v.Description,
						Parameters:  genAiPSchema,
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

func (d *Ai) SetFunctions(functions frags.Functions) {
	d.Functions = functions
}

func joinParts(parts []*genai.Part) string {
	out := ""
	for _, part := range parts {
		out += part.Text
	}
	return out
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
