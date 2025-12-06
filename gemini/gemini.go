package gemini

import (
	"context"

	"github.com/jinzhu/copier"
	"github.com/theirish81/frags"
	"google.golang.org/genai"
)

// Gemini defaults
var temperature float32 = 0.1
var topK float32 = 40
var topP float32 = 0.9

const jsonContentType = "application/json"
const textContentType = "text/plain"
const defaultModel = "gemini-2.5-flash"

// Ai is a wrapper around the genai client for Frags
type Ai struct {
	client    *genai.Client
	content   []*genai.Content
	modelName string
	Functions frags.Functions
}

// NewAI creates a new Ai wrapper
func NewAI(client *genai.Client) *Ai {
	return &Ai{
		client:    client,
		content:   make([]*genai.Content, 0),
		modelName: defaultModel,
		Functions: frags.Functions{},
	}
}

// New creates a new Ai instance and returns it
func (d *Ai) New() frags.Ai {
	return &Ai{
		client:    d.client,
		content:   make([]*genai.Content, 0),
		modelName: d.modelName,
		Functions: d.Functions,
	}
}

// Ask performs a query against the Gemini API, according to the Frags interface
func (d *Ai) Ask(ctx context.Context, text string, schema *frags.Schema, tools frags.Tools, resources ...frags.ResourceData) ([]byte, error) {
	parts := make([]*genai.Part, 0)
	for _, resource := range resources {
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
	if err != nil {
		return nil, err
	}
	cfg := genai.GenerateContentConfig{
		ResponseMIMEType: ct,
		ResponseSchema:   genAiSchema,
		Temperature:      &temperature,
		TopK:             &topK,
		TopP:             &topP,
		Tools:            tx,
	}
	keepGoing := true
	out := ""
	d.content = append(d.content, newMsg)
	for keepGoing {
		res, err := d.client.Models.GenerateContent(ctx, d.modelName, d.content, &cfg)
		if err != nil {
			return nil, err
		}
		d.content = append(d.content, res.Candidates[0].Content)
		if res.FunctionCalls() != nil {
			for _, fc := range res.FunctionCalls() {
				d.content = append(d.content, genai.NewContentFromFunctionCall(fc.Name, fc.Args, genai.RoleModel))
				fres, ferr := d.Functions[fc.Name].Func(fc.Args)
				if ferr != nil {
					return nil, ferr
				} else {
					d.content = append(d.content, genai.NewContentFromFunctionResponse(fc.Name, fres, genai.RoleUser))
				}
			}
			keepGoing = true
		} else {
			keepGoing = false
			d.content = append(d.content, res.Candidates[0].Content)
			out = joinParts(res.Candidates[0].Content.Parts)
		}
	}
	return []byte(out), nil
}

func (d *Ai) configureTools(tools frags.Tools) ([]*genai.Tool, error) {
	tx := make([]*genai.Tool, 0)
	fd := make([]*genai.FunctionDeclaration, 0)
	for _, tool := range tools {
		if tool.Type == frags.ToolTypeFunction {
			if fx, found := d.Functions[tool.Name]; found {
				pSchema := fx.Schema
				if tool.Parameters != nil {
					pSchema = tool.Parameters
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
		}
	}
	if len(fd) > 0 {
		tx = append(tx, &genai.Tool{
			FunctionDeclarations: fd,
		})
	}

	if tools.HasType(frags.ToolTypeInternetSearch) {
		tx = append(tx, &genai.Tool{
			GoogleSearch: &genai.GoogleSearch{},
		})
	}
	return tx, nil
}

func joinParts(parts []*genai.Part) string {
	out := ""
	for _, part := range parts {
		out += part.Text
	}
	return out
}
