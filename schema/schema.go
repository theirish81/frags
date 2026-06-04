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

package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"gopkg.in/yaml.v3"
)

const Object = "object"
const String = "string"
const Integer = "integer"
const Number = "number"
const Boolean = "boolean"
const Array = "array"

type Type string

// Schema represents a JSON schema with x-session extensions.
type Schema struct {
	OneOf            []*Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf            []*Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Default          any                `json:"default,omitempty" yaml:"default,omitempty"`
	Description      string             `json:"description,omitempty" yaml:"description,omitempty"`
	Enum             []any              `json:"enum,omitempty" yaml:"enum,omitempty"`
	Example          any                `json:"example,omitempty" yaml:"example,omitempty"`
	Format           string             `json:"format,omitempty" yaml:"format,omitempty"`
	Items            *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	MaxItems         *int64             `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MaxLength        *int64             `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MaxProperties    *int64             `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	Maximum          *float64           `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	MinItems         *int64             `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MinLength        *int64             `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MinProperties    *int64             `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Minimum          *float64           `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Nullable         *bool              `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	Pattern          string             `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Properties       map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	PropertyOrdering []string           `json:"propertyOrdering,omitempty" yaml:"propertyOrdering,omitempty"`
	Required         []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Title            string             `json:"title,omitempty" yaml:"title,omitempty"`
	Type             Type               `json:"type,omitempty" yaml:"type,omitempty"`
	XSession         *string            `json:"x-session,omitempty" yaml:"x-session,omitempty"`
	Ref              *string            `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	XUI              map[string]any     `json:"-" yaml:"-"`
}

func FromAny(data any) (*Schema, error) {
	schema := Schema{}
	if data == nil {
		return nil, nil
	}
	switch typed := data.(type) {
	case string:
		err := json.Unmarshal([]byte(typed), &schema)
		return &schema, err
	case map[string]any:
		err := mapstructure.Decode(typed, &schema)
		return &schema, err
	case []byte:
		err := json.Unmarshal(typed, &schema)
		return &schema, err
	case json.RawMessage:
		if err := json.Unmarshal(typed, &schema); err != nil {
			return &schema, err
		}
	default:
		err := schema.CopyFrom(typed)
		return &schema, err
	}
	return &schema, errors.New("cannot convert schema")
}

// FromYAML unmarshals a YAML document into the Schema.
func (s *Schema) FromYAML(data []byte) error {
	return yaml.Unmarshal(data, s)
}

// GetSessionsIDs returns the IDs of all sessions in the schema.
func (s *Schema) GetSessionsIDs() []string {
	sessions := make([]string, 0)
	for _, v := range s.Properties {
		if v.XSession != nil {
			if !slices.Contains(sessions, *v.XSession) {
				sessions = append(sessions, *v.XSession)
			}
		}
	}
	return sessions
}

// GetSession returns a Schema for a specific session.
func (s *Schema) GetSession(sessionID string) (*Schema, error) {
	clonedSchema := *s
	if !slices.Contains(s.GetSessionsIDs(), sessionID) {
		return nil, errors.New("sessionID not found")
	}
	px := make(map[string]*Schema)
	req := make([]string, 0)
	for k, v := range clonedSchema.Properties {
		if v.XSession != nil && *v.XSession == sessionID {
			px[k] = v
			if slices.Contains(clonedSchema.Required, k) {
				req = append(req, k)
			}
		}
	}
	clonedSchema.Properties = px
	clonedSchema.Required = req
	return &clonedSchema, nil
}

// Resolve resolves all the references in the schema.
func (s *Schema) Resolve(schemas map[string]Schema) error {
	return s.resolve(s, schemas, make(map[string]bool))
}

// resolve resolves all the references in the schema (recursive function)
func (s *Schema) resolve(schema *Schema, schemas map[string]Schema, visited map[string]bool) error {
	if schema == nil {
		return nil
	}
	if schema.Ref != nil {
		ref := *schema.Ref
		if visited[ref] {
			return nil
		}
		if strings.HasPrefix(ref, "#/components/schemas/") {
			visited[ref] = true
			defer func() { delete(visited, ref) }()
			schemaName := strings.TrimPrefix(ref, "#/components/schemas/")
			if resolvedSchema, ok := schemas[schemaName]; ok {
				originalXSession := schema.XSession

				*schema = resolvedSchema

				schema.XSession = originalXSession
				schema.Ref = nil

				if err := s.resolve(schema, schemas, visited); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("schema not found: %s", ref)
			}
		}
	}

	if schema.Properties != nil {
		for _, propSchema := range schema.Properties {
			if err := s.resolve(propSchema, schemas, visited); err != nil {
				return err
			}
		}
	}

	if schema.Items != nil {
		if err := s.resolve(schema.Items, schemas, visited); err != nil {
			return err
		}
	}

	if schema.AnyOf != nil {
		for _, anyOfSchema := range schema.AnyOf {
			if err := s.resolve(anyOfSchema, schemas, visited); err != nil {
				return err
			}
		}
	}
	if schema.OneOf != nil {
		for _, anyOfSchema := range schema.OneOf {
			if err := s.resolve(anyOfSchema, schemas, visited); err != nil {
				return err
			}
		}
	}

	return nil
}
