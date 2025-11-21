package ollama

import "github.com/theirish81/frags"

// Message represents a message sent to the LLM.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request represents a request to Ollama
type Request struct {
	Model    string       `json:"model"`
	Messages []Message    `json:"messages"`
	Format   frags.Schema `json:"format,omitempty"`
	Stream   bool         `json:"stream"`
}

// Response represents a response from Ollama
type Response struct {
	Message Message `json:"message"`
}
