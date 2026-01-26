package ollama

import (
	"github.com/theirish81/frags/schema"
)

// Message represents a message sent to the LLM.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Request represents a request to Ollama
type Request struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Format   *schema.Schema   `json:"format,omitempty"`
	Stream   bool             `json:"stream"`
	Think    bool             `json:"think"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
	Options  Options          `json:"options,omitempty"`
}

type ToolDefinition struct {
	Type     string      `json:"type" yaml:"type"`
	Function FunctionDef `json:"function" yaml:"function"`
}

func (t ToolDefinition) String() string {
	return t.Function.Name
}

// FunctionDef represents a function definition
type FunctionDef struct {
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	Parameters  *schema.Schema `json:"parameters" yaml:"parameters"`
}

// Response represents a response from Ollama
type Response struct {
	Message Message `json:"message"`
}

type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type Options struct {
	NumPredict        int      `json:"num_predict,omitempty"`
	Stop              []string `json:"stop,omitempty"`
	Temperature       float32  `json:"temperature,omitempty"`
	TopK              float32  `json:"top_k,omitempty"`
	TopP              float32  `json:"top_p,omitempty"`
	RepetitionPenalty float32  `json:"repeat_penalty,omitempty"`
}
