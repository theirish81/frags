package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
	Input              []InputItem   `json:"input"`
	Text               *Text         `json:"text,omitempty"`
	Tools              []ChatGptTool `json:"tools,omitempty"`
	Modalities         []string      `json:"modalities,omitempty"`
	PreviousResponseID string        `json:"previous_response_id,omitempty"`
}

// InputItem represents an input item in the Responses API
type InputItem struct {
	Role      string        `json:"role,omitempty"`
	CallID    string        `json:"call_id,omitempty"`
	Content   []ContentPart `json:"content,omitempty"`
	Type      string        `json:"type,omitempty"`
	Name      string        `json:"name,omitempty"`
	Arguments *ArgsUnion    `json:"arguments,omitempty"`
	Output    any           `json:"output,omitempty"`
}

// ContentPart represents a part of content (text, image, file)
type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	FileID   string `json:"file_id,omitempty"`
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
	ID         string      `json:"id"`
	Object     string      `json:"object"`
	Created    int64       `json:"created"`
	Model      string      `json:"model"`
	Output     []InputItem `json:"output"`
	OutputText string      `json:"output_text,omitempty"`
	Usage      Usage       `json:"usage"`
}

func (r Response) HasFunctionCalls() bool {
	for _, item := range r.Output {
		if item.Type == "function_call" {
			return true
		}
	}
	return false
}

func (r Response) FunctionCalls() []InputItem {
	items := make([]InputItem, 0)
	for _, item := range r.Output {
		if item.Type == "function_call" {
			items = append(items, item)
		}
	}
	return items
}

// Usage represents token usage information
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ChatOptions contains optional parameters for the Chat method
type ChatOptions struct {
	Schema          any
	FileIDs         []string
	Tools           []ChatGptTool
	EnableWebSearch bool
}

// Chat sends a message with optional file IDs, schema, and tools, and receives a response
func (c *Client) Chat(message string, opts *ChatOptions) (*Response, error) {
	if opts == nil {
		opts = &ChatOptions{}
	}

	// Build the content array
	content := []ContentPart{
		{
			Type: "input_text",
			Text: message,
		},
	}

	// Add file IDs if present
	for _, fileID := range opts.FileIDs {
		content = append(content, ContentPart{
			Type:   "input_file",
			FileID: fileID,
		})
	}

	req := ResponseRequest{
		Model: "gpt-4o",
		Input: []InputItem{
			{
				Role:    "user",
				Content: content,
			},
		},
	}

	// Configure response format if schema is provided
	if opts.Schema != nil {
		req.Text = &Text{
			Format: &ResponseFormat{
				Name:   "response",
				Type:   "json_schema",
				Schema: opts.Schema,
			},
		}
	}

	// Add tools if provided
	if len(opts.Tools) > 0 {
		req.Tools = opts.Tools
	}

	// Add web search tool if enabled
	if opts.EnableWebSearch {
		req.Tools = append(req.Tools, ChatGptTool{
			Type: "web_search_20250305",
			Name: "web_search",
		})
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/responses", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

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

	return &response, nil
}

// NewWebSearchTool creates a web search tool
func NewWebSearchTool() ChatGptTool {
	return ChatGptTool{
		Type: "web_search_20250305",
		Name: "web_search",
	}
}

// NewFunctionTool creates a custom function tool
func NewFunctionTool(name, description string, parameters map[string]interface{}) ChatGptTool {
	return ChatGptTool{
		Type:        "function",
		Name:        name,
		Description: description,
		Parameters:  parameters,
	}
}

// convertToMap converts any type to map[string]interface{}
func convertToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}
