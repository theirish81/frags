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
package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	// Pre-process the "type" field: it may be a string or an array of strings.
	// We resolve it to a single string here, before the alias decode sees it,
	// so that Type stays a plain string throughout the rest of the codebase.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if typeRaw, ok := raw["type"]; ok {
		var single string
		if err := json.Unmarshal(typeRaw, &single); err != nil {
			// Not a string — try array.
			var arr []string
			if err := json.Unmarshal(typeRaw, &arr); err != nil {
				return fmt.Errorf("schema type: expected string or array of strings: %w", err)
			}
			for _, t := range arr {
				if t == "null" {
					tr := true
					s.Nullable = &tr
				} else if s.Type == "" {
					s.Type = Type(t)
				}
			}
			// Replace the array with the resolved scalar so the alias decode
			// below can handle it normally as a string.
			resolved, err := json.Marshal(s.Type)
			if err != nil {
				return err
			}
			raw["type"] = resolved
			data, err = json.Marshal(raw)
			if err != nil {
				return err
			}
		}
	}

	if err := json.Unmarshal(data, (*schemaAlias)(s)); err != nil {
		return err
	}

	// Many LLMs will not like a type: array with a missing "items" and explode with fireworks.
	// The explosion will happen as soon as the LLM sees it, so it's a blocker.
	// Given this is a rare event, we decided to cover that case and convert it to items -> type: string
	// Not perfect, but close enough, I guess.
	if s.Type == "array" && s.Items == nil {
		s.Items = &Schema{Type: "string"}
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
	// Pre-process the "type" field: it may be a scalar or a sequence of strings.
	// We resolve it to a single scalar here, before the alias decode sees it,
	// so that Type stays a plain string throughout the rest of the codebase.
	if value.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(value.Content); i += 2 {
			if value.Content[i].Value != "type" {
				continue
			}
			val := value.Content[i+1]
			if val.Kind == yaml.SequenceNode {
				for _, item := range val.Content {
					if item.Value == "null" {
						tr := true
						s.Nullable = &tr
					} else if s.Type == "" {
						s.Type = Type(item.Value)
					}
				}
				// Replace the sequence node with a plain scalar so the
				// alias decode below sees a normal string.
				value.Content[i+1] = &yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: string(s.Type),
				}
			}
			break
		}
	}

	if err := value.Decode((*schemaAlias)(s)); err != nil {
		return err
	}
	// Many LLMs will not like a type: array with a missing "items" and explode with fireworks.
	// The explosion will happen as soon as the LLM sees it, so it's a blocker.
	// Given this is a rare event, we decided to cover that case and convert it to items -> type: string
	// Not perfect, but close enough, I guess.
	if s.Type == "array" && s.Items == nil {
		s.Items = &Schema{Type: "string"}
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
