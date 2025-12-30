package chatgpt

import (
	"net/http"
	"time"

	"github.com/theirish81/frags"
)

// Client represents the client for OpenAI's Responses API
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new instance of the OpenAI Responses client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type Text struct {
	Format *ResponseFormat `json:"format,omitempty"`
}

// ResponseRequest represents a request to the Responses API
type ResponseRequest struct {
	Model              string        `json:"model"`
	Input              Messages      `json:"input"`
	Text               *Text         `json:"text,omitempty"`
	Tools              []ChatGptTool `json:"tools,omitempty"`
	Modalities         []string      `json:"modalities,omitempty"`
	PreviousResponseID string        `json:"previous_response_id,omitempty"`
}

func NewResponseRequest(model string, input []Message, tools []ChatGptTool, schema *frags.Schema) ResponseRequest {
	req := ResponseRequest{
		Model: model,
		Input: input,
		Tools: tools,
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
	return req
}

// Message represents an input item in the Responses API
type Message struct {
	Role      string       `json:"role,omitempty"`
	CallID    string       `json:"call_id,omitempty"`
	Content   ContentParts `json:"content,omitempty"`
	Type      string       `json:"type,omitempty"`
	Name      string       `json:"name,omitempty"`
	Arguments *ArgsUnion   `json:"arguments,omitempty"`
	Output    any          `json:"output,omitempty"`
}

func NewUserMessage(text string) Message {
	return Message{
		Role: "user",
		Content: ContentParts{
			{
				Type: "input_text",
				Text: text,
			},
		},
	}
}

type Messages []Message

func (m Messages) Last() Message {
	return m[len(m)-1]
}

// ContentPart represents a part of content (text, image, file)
type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	FileID   string `json:"file_id,omitempty"`
}

type ContentParts []ContentPart

func (c ContentParts) First() ContentPart {
	return c[0]
}

// ResponseFormat specifies the response format with JSON schema
type ResponseFormat struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Schema any    `json:"schema,omitempty"`
}

// ChatGptTool represents a tool that can be used by the model
type ChatGptTool struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// Response represents the response from the Responses API
type Response struct {
	ID         string   `json:"id"`
	Object     string   `json:"object"`
	Created    int64    `json:"created"`
	Model      string   `json:"model"`
	Output     Messages `json:"output"`
	OutputText string   `json:"output_text,omitempty"`
}

func (r Response) HasFunctionCalls() bool {
	for _, item := range r.Output {
		if item.Type == "function_call" {
			return true
		}
	}
	return false
}

func (r Response) FunctionCalls() []Message {
	items := make([]Message, 0)
	for _, item := range r.Output {
		if item.Type == "function_call" {
			items = append(items, item)
		}
	}
	return items
}

// ChatOptions contains optional parameters for the Chat method
type ChatOptions struct {
	Schema          any
	FileIDs         []string
	Tools           []ChatGptTool
	EnableWebSearch bool
}
