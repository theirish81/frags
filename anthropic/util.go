/*
 * Copyright (C) 2026 Simone Pezzano
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

package anthropic

import (
	"encoding/json"

	"github.com/theirish81/frags/schema"
)

// SchemaToClaudeMap converts a Schema to a map[string]any suitable for the Claude API.
// It recursively processes all nested schemas and automatically injects
// `additionalProperties: false` on any object-typed layer, as required by Claude.
func SchemaToClaudeMap(s *schema.Schema) map[string]any {
	if s == nil {
		return nil
	}

	m := map[string]any{}

	// Scalar / metadata fields
	if s.Type != "" {
		m["type"] = s.Type
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if s.Title != "" {
		m["title"] = s.Title
	}
	if s.Format != "" {
		m["format"] = s.Format
	}
	if s.Pattern != "" {
		m["pattern"] = s.Pattern
	}
	if s.Default != nil {
		m["default"] = s.Default
	}
	if s.Example != nil {
		m["example"] = s.Example
	}
	// $ref: dropped — not supported by Claude structured outputs

	// Numeric constraints
	// minimum, maximum, minLength, maxLength, minItems, maxItems: dropped — not supported
	if s.MinProperties != nil {
		m["minProperties"] = *s.MinProperties
	}
	if s.MaxProperties != nil {
		m["maxProperties"] = *s.MaxProperties
	}

	// Nullable
	if s.Nullable != nil {
		m["nullable"] = *s.Nullable
	}

	// Enum
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}

	// Required
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}

	// PropertyOrdering (Gemini-style extension, kept for compatibility)
	if len(s.PropertyOrdering) > 0 {
		m["propertyOrdering"] = s.PropertyOrdering
	}

	// Items (arrays)
	if s.Items != nil {
		m["items"] = SchemaToClaudeMap(s.Items)
	}

	// Properties (objects) — recurse into each value
	if len(s.Properties) > 0 {
		props := make(map[string]any, len(s.Properties))
		for k, v := range s.Properties {
			props[k] = SchemaToClaudeMap(v)
		}
		m["properties"] = props
	}

	// Inject additionalProperties: false on object layers, as required by Claude
	if s.Type == "object" {
		m["additionalProperties"] = false
	}

	// Combiners
	if len(s.OneOf) > 0 {
		oneOf := make([]any, len(s.OneOf))
		for i, sub := range s.OneOf {
			oneOf[i] = SchemaToClaudeMap(sub)
		}
		m["oneOf"] = oneOf
	}
	if len(s.AnyOf) > 0 {
		anyOf := make([]any, len(s.AnyOf))
		for i, sub := range s.AnyOf {
			anyOf[i] = SchemaToClaudeMap(sub)
		}
		m["anyOf"] = anyOf
	}

	return m
}

func decodeRawJsonMessageToMap(input json.RawMessage) map[string]any {
	inputMap := make(map[string]any)
	if input == nil || len(input) == 0 {
		return inputMap
	}
	inputBytes, _ := json.Marshal(input)
	_ = json.Unmarshal(inputBytes, &inputMap)
	return inputMap
}

type FunctionResponseMap map[string]any

func NewFunctionResponseMap(data any, err error) FunctionResponseMap {
	res := FunctionResponseMap{
		"output": data,
	}
	if err != nil {
		res["error"] = err.Error()
	}
	res["system_guidance"] = `
* in case of success, ignore this guidance
* if recoverable failure, retry function calling MAX 3 times per function, adjusting the parameters as necessary.
* if the error is deemed not solvable, not recoverable or we hit the 3rd attempt, respond with the failure reasons, preceded by the flag [FATAL]`
	return res
}
