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

// Ai is a wrapper around the genai client for Frags
type Ai struct {
	client    *genai.Client
	content   []*genai.Content
	modelName string
}

// NewAI creates a new Ai wrapper
func NewAI(client *genai.Client) *Ai {
	return &Ai{
		client:    client,
		content:   make([]*genai.Content, 0),
		modelName: "gemini-2.5-flash",
	}
}

// New creates a new Ai instance and returns it
func (d *Ai) New() frags.Ai {
	return &Ai{
		client:    d.client,
		content:   make([]*genai.Content, 0),
		modelName: d.modelName,
	}
}

// Ask performs a query against the Gemini API, according to the Frags interface
func (d *Ai) Ask(ctx context.Context, text string, schema frags.Schema, resources ...frags.Resource) ([]byte, error) {
	parts := make([]*genai.Part, 0)
	for _, resource := range resources {
		parts = append(parts, genai.NewPartFromBytes(resource.Data, resource.MediaType))
	}
	parts = append(parts, genai.NewPartFromText(text))
	genAiSchema := genai.Schema{}
	if err := copier.Copy(&genAiSchema, &schema); err != nil {
		return nil, err
	}
	newMsg := genai.NewContentFromParts(parts, genai.RoleUser)

	cfg := genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   &genAiSchema,
		Temperature:      &temperature,
		TopK:             &topK,
		TopP:             &topP,
	}
	d.content = append(d.content, newMsg)
	res, err := d.client.Models.GenerateContent(ctx, d.modelName, d.content, &cfg)
	if err != nil {
		return nil, err
	}
	d.content = append(d.content, res.Candidates[0].Content)
	out := joinParts(res.Candidates[0].Content.Parts)
	return []byte(out), nil
}

func joinParts(parts []*genai.Part) string {
	out := ""
	for _, part := range parts {
		out += part.Text
	}
	return out
}
