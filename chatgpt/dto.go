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
	"bytes"
	"encoding/json"

	"github.com/theirish81/frags/schema"
)

const PartTypeInputText = "input_text"
const PartTypeFunctionCall = "function_call"
const PartTypeFunctionCallOutput = "function_call_output"
const PartTypeJsonSchema = "json_schema"
const PartTypeInputFile = "input_file"
const PartTypeFunction = "function"
const RoleUser = "user"

type Text struct {
	Format *ResponseFormat `json:"format,omitempty"`
}

type ReasoningConfig struct {
	Effort string `json:"effort,omitempty"`
}

// ResponseRequest represents a request to the Responses API
type ResponseRequest struct {
	Model              string           `json:"model"`
	Instructions       string           `json:"instructions"`
	Input              Messages         `json:"input"`
	Text               *Text            `json:"text,omitempty"`
	Tools              []ChatGptTool    `json:"tools,omitempty"`
	Modalities         []string         `json:"modalities,omitempty"`
	PreviousResponseID string           `json:"previous_response_id,omitempty"`
	Reasoning          *ReasoningConfig `json:"reasoning,omitempty"`
}

func NewResponseRequest(model string, input []Message, instructions string, tools []ChatGptTool, schema *schema.Schema) ResponseRequest {
	req := ResponseRequest{
		Model:        model,
		Input:        input,
		Tools:        tools,
		Instructions: instructions,
	}
	if schema != nil {
		req.Text = &Text{
			Format: &ResponseFormat{
				Name:   "response",
				Type:   PartTypeJsonSchema,
				Schema: schema,
			},
		}
	}
	return req
}

// Message represents an input item in the Responses API
type Message struct {
	ID               string       `json:"id,omitempty"`
	Role             string       `json:"role,omitempty"`
	CallID           string       `json:"call_id,omitempty"`
	Content          ContentParts `json:"content,omitempty"`
	Type             string       `json:"type,omitempty"`
	Name             string       `json:"name,omitempty"`
	Arguments        *ArgsUnion   `json:"arguments,omitempty"`
	Output           any          `json:"output,omitempty"`
	Status           string       `json:"status,omitempty"`
	Summary          any          `json:"summary,omitempty"`
	DurationMs       any          `json:"duration_ms,omitempty"`
	EncryptedContent string       `json:"encrypted_content,omitempty"`
	Action           any          `json:"action,omitempty"`
	Results          any          `json:"results,omitempty"`
}

func NewUserMessage(text string) Message {
	return Message{
		Role: RoleUser,
		Content: ContentParts{
			{
				Type: PartTypeInputText,
				Text: text,
			},
		},
	}
}

type Messages []Message

func (m *Messages) Last() Message {
	return (*m)[len(*m)-1]
}

// ContentPart represents a part of content (text, image, file)
type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"-"`
	ImageURL string `json:"image_url,omitempty"`
	FileID   string `json:"file_id,omitempty"`
	TextObj  any    `json:"-"`
}

func (c *ContentPart) UnmarshalJSON(data []byte) error {
	type Alias ContentPart
	var aux struct {
		Alias
		Text json.RawMessage `json:"text,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.Type = aux.Type
	c.ImageURL = aux.ImageURL
	c.FileID = aux.FileID

	if len(aux.Text) > 0 {
		aux.Text = bytes.TrimSpace(aux.Text)
		if len(aux.Text) > 0 && aux.Text[0] == '{' {
			// It is a structured text object (containing value and annotations)
			var obj any
			if err := json.Unmarshal(aux.Text, &obj); err != nil {
				return err
			}
			c.TextObj = obj

			// Extract the value as string for Text field backwards compatibility
			var textVal struct {
				Value string `json:"value"`
			}
			_ = json.Unmarshal(aux.Text, &textVal)
			c.Text = textVal.Value
		} else {
			// It is a plain string
			var s string
			if err := json.Unmarshal(aux.Text, &s); err != nil {
				return err
			}
			c.Text = s
		}
	}
	return nil
}

func (c ContentPart) MarshalJSON() ([]byte, error) {
	type Alias ContentPart

	// If this is a non-text type and contains no text content, serialize without 'text' field entirely.
	isTextType := c.Type == "text" || c.Type == "input_text" || c.Type == "output_text" || c.Type == ""
	if !isTextType && c.TextObj == nil && c.Text == "" {
		return json.Marshal(Alias(c))
	}

	var textVal any = c.Text
	if c.TextObj != nil {
		textVal = c.TextObj
	}

	return json.Marshal(struct {
		Alias
		Text any `json:"text"`
	}{
		Alias: Alias(c),
		Text:  textVal,
	})
}

type ContentParts []ContentPart

func (c *ContentParts) InsertTextPart(text string) {
	*c = append([]ContentPart{{
		Type: PartTypeInputText,
		Text: text,
	}}, *c...)
}

func (c *ContentParts) InsertFileMessage(fileId string) {
	*c = append([]ContentPart{{
		Type:   PartTypeInputFile,
		FileID: fileId,
	}}, *c...)
}

func (c *ContentParts) First() ContentPart {
	return (*c)[0]
}

// ResponseFormat specifies the response format with JSON schema
type ResponseFormat struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Strict bool   `json:"strict"`
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
		if item.Type == PartTypeFunctionCall {
			return true
		}
	}
	return false
}

func (r Response) FunctionCalls() []Message {
	items := make([]Message, 0)
	for _, item := range r.Output {
		if item.Type == PartTypeFunctionCall {
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

type FileDescriptor struct {
	Id        string `json:"id"`
	Object    string `json:"object"`
	Bytes     int    `json:"bytes"`
	CreatedAt int    `json:"created_at"`
	ExpiresAt int    `json:"expires_at"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
}
