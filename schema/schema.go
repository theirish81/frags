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
	"sort"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/jinzhu/copier"
	"gopkg.in/yaml.v3"
)

const Object = "object"
const String = "string"
const Integer = "integer"
const Number = "number"
const Boolean = "boolean"
const Array = "array"

// Schema represents a JSON schema with x-phase and x-session extensions.
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
	Type             string             `json:"type,omitempty" yaml:"type,omitempty"`
	XPhase           int                `json:"x-phase,omitempty" yaml:"x-phase,omitempty"`
	XSession         *string            `json:"x-session,omitempty" yaml:"x-session,omitempty"`
	Ref              *string            `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	XUI              map[string]any     `json:"-" yaml:"-"`
}

type schemaAlias Schema

func (s Schema) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(schemaAlias(s))
	if err != nil {
		return nil, err
	}

	if len(s.XUI) == 0 {
		return b, nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	for k, v := range s.XUI {
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("x-ui-%s: %w", k, err)
		}
		m["x-ui-"+k] = raw
	}
	return json.Marshal(m)
}

func (s *Schema) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, (*schemaAlias)(s)); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		if !strings.HasPrefix(k, "x-ui-") {
			continue
		}
		suffix := strings.TrimPrefix(k, "x-ui-")
		var val any
		if err := json.Unmarshal(v, &val); err != nil {
			return fmt.Errorf("%s: %w", k, err)
		}
		if s.XUI == nil {
			s.XUI = make(map[string]any)
		}
		s.XUI[suffix] = val
	}
	return nil
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
		err := copier.Copy(&schema, typed)
		return &schema, err
	}
	return &schema, errors.New("cannot convert schema")
}

func (s Schema) MarshalYAML() (any, error) {
	type kv struct {
		key string
		val any
	}

	var pairs []kv

	add := func(key string, val any) {
		pairs = append(pairs, kv{key, val})
	}

	if len(s.OneOf) > 0 {
		add("oneOf", s.OneOf)
	}
	if len(s.AnyOf) > 0 {
		add("anyOf", s.AnyOf)
	}
	if s.Default != nil {
		add("default", s.Default)
	}
	if s.Description != "" {
		add("description", s.Description)
	}
	if len(s.Enum) > 0 {
		add("enum", s.Enum)
	}
	if s.Example != nil {
		add("example", s.Example)
	}
	if s.Format != "" {
		add("format", s.Format)
	}
	if s.Items != nil {
		add("items", s.Items)
	}
	if s.MaxItems != nil {
		add("maxItems", s.MaxItems)
	}
	if s.MaxLength != nil {
		add("maxLength", s.MaxLength)
	}
	if s.MaxProperties != nil {
		add("maxProperties", s.MaxProperties)
	}
	if s.Maximum != nil {
		add("maximum", s.Maximum)
	}
	if s.MinItems != nil {
		add("minItems", s.MinItems)
	}
	if s.MinLength != nil {
		add("minLength", s.MinLength)
	}
	if s.MinProperties != nil {
		add("minProperties", s.MinProperties)
	}
	if s.Minimum != nil {
		add("minimum", s.Minimum)
	}
	if s.Nullable != nil {
		add("nullable", s.Nullable)
	}
	if s.Pattern != "" {
		add("pattern", s.Pattern)
	}
	if len(s.Properties) > 0 {
		add("properties", s.Properties)
	}
	if len(s.PropertyOrdering) > 0 {
		add("propertyOrdering", s.PropertyOrdering)
	}
	if len(s.Required) > 0 {
		add("required", s.Required)
	}
	if s.Title != "" {
		add("title", s.Title)
	}
	if s.Type != "" {
		add("type", s.Type)
	}
	if s.XPhase != 0 {
		add("x-phase", s.XPhase)
	}
	if s.XSession != nil {
		add("x-session", s.XSession)
	}
	if s.Ref != nil {
		add("$ref", s.Ref)
	}

	for k, v := range s.XUI {
		add("x-ui-"+k, v)
	}

	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, p := range pairs {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: p.key}
		valNode := &yaml.Node{}
		if err := valNode.Encode(p.val); err != nil {
			return nil, fmt.Errorf("yaml marshal key %q: %w", p.key, err)
		}
		if valNode.Kind == yaml.DocumentNode && len(valNode.Content) > 0 {
			valNode = valNode.Content[0]
		}
		node.Content = append(node.Content, keyNode, valNode)
	}
	return node, nil
}

func (s *Schema) UnmarshalYAML(value *yaml.Node) error {
	if err := value.Decode((*schemaAlias)(s)); err != nil {
		return err
	}

	if value.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(value.Content); i += 2 {
		key := value.Content[i].Value
		if !strings.HasPrefix(key, "x-ui-") {
			continue
		}
		suffix := strings.TrimPrefix(key, "x-ui-")
		var val any
		if err := value.Content[i+1].Decode(&val); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if s.XUI == nil {
			s.XUI = make(map[string]any)
		}
		s.XUI[suffix] = val
	}
	return nil
}

// FromYAML unmarshals a YAML document into the Schema.
func (s *Schema) FromYAML(data []byte) error {
	return yaml.Unmarshal(data, s)
}

// GetPhase returns a Schema for a specific phase.
func (s *Schema) GetPhase(phase int) (Schema, error) {
	clonedSchema := *s
	if !slices.Contains(s.GetPhaseIndexes(), phase) {
		return clonedSchema, errors.New("phase not found")
	}
	px := make(map[string]*Schema)
	req := make([]string, 0)
	for k, v := range clonedSchema.Properties {
		if v.XPhase == phase {
			px[k] = v
			if slices.Contains(clonedSchema.Required, k) {
				req = append(req, k)
			}
		}
	}
	clonedSchema.Properties = px
	clonedSchema.Required = req
	return clonedSchema, nil
}

// GetPhaseIndexes returns the indexes of all phases in the schema.
func (s *Schema) GetPhaseIndexes() []int {
	idx := make([]int, 0)
	for _, v := range s.Properties {
		if !slices.Contains(idx, v.XPhase) {
			idx = append(idx, v.XPhase)
		}
	}
	sort.Ints(idx)
	return idx
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
func (s *Schema) GetSession(sessionID string) (Schema, error) {
	clonedSchema := *s
	if !slices.Contains(s.GetSessionsIDs(), sessionID) {
		return clonedSchema, errors.New("sessionID not found")
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
	return clonedSchema, nil
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
				originalXPhase := schema.XPhase
				originalXSession := schema.XSession

				*schema = resolvedSchema

				schema.XPhase = originalXPhase
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
