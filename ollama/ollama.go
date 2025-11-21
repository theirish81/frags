package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/theirish81/frags"
)

// Ai implements the Ai interface for Ollama.
type Ai struct {
	client   *http.Client
	baseURL  string
	model    string
	messages []Message
}

// NewAI creates a new Ai instance
func NewAI(baseURL string, model string) *Ai {
	return &Ai{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 10,
			},
		},
		messages: make([]Message, 0),
	}
}

// New creates a new Ai instance and returns it
func (d *Ai) New() frags.Ai {
	return &Ai{
		client:   d.client,
		baseURL:  d.baseURL,
		model:    d.model,
		messages: make([]Message, 0),
	}
}

// Ask performs a query against the Ollama API, according to the Frags interface
func (d *Ai) Ask(ctx context.Context, text string, schema frags.Schema, resources ...frags.ResourceData) ([]byte, error) {
	message := Message{
		Content: "",
		Role:    "user",
	}
	for _, r := range resources {
		if r.MediaType != "text/plain" {
			return nil, errors.New("ollama only supports text resources")
		}
		message.Content += message.Content + " === " + r.Identifier + " === \n" + string(r.Data) + "\n"
	}
	message.Content += "\n" + text
	d.messages = append(d.messages, message)
	request := Request{
		Messages: d.messages,
		Model:    d.model,
		Format:   schema,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(requestBody)
	apiUrl, err := url.Parse(d.baseURL + "/api/chat")
	if err != nil {
		return nil, err
	}
	req := http.Request{
		Method: "POST",
		URL:    apiUrl,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(reader),
	}
	response, err := d.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		if response.Body != nil {
			_ = response.Body.Close()
		}
	}()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	responseMessage := Response{}
	d.messages = append(d.messages, responseMessage.Message)
	err = json.Unmarshal(responseBody, &responseMessage)
	return []byte(responseMessage.Message.Content), err
}
