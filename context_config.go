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

package frags

import (
	"encoding/json"
	"fmt"

	"github.com/theirish81/frags/evaluators"
	"gopkg.in/yaml.v3"
)

type ContextConfig struct {
	Bool   *bool
	String *string
}

func (c *ContextConfig) IsTrue() bool {
	return c.Bool != nil && *c.Bool
}

func (c *ContextConfig) HasTemplate() bool {
	return c.String != nil
}

func (c *ContextConfig) RenderTemplate(scope evaluators.EvalScope) (string, error) {
	if c.String == nil {
		return "", nil
	}
	return evaluators.EvaluateTemplate(*c.String, scope)
}

// --- JSON ---

func (c *ContextConfig) MarshalJSON() ([]byte, error) {
	if c.Bool != nil {
		return json.Marshal(*c.Bool)
	}
	if c.String != nil {
		return json.Marshal(*c.String)
	}
	return []byte("null"), nil
}

func (c *ContextConfig) UnmarshalJSON(data []byte) error {
	// Try bool first
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		c.Bool = &b
		return nil
	}
	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		c.String = &s
		return nil
	}
	return fmt.Errorf("ContextConfig: cannot unmarshal JSON value: %s", data)
}

// --- YAML ---

func (c *ContextConfig) MarshalYAML() (interface{}, error) {
	if c.Bool != nil {
		return *c.Bool, nil
	}
	if c.String != nil {
		return *c.String, nil
	}
	return nil, nil
}

func (c *ContextConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Tag == "!!bool" {
		var b bool
		if err := value.Decode(&b); err != nil {
			return err
		}
		c.Bool = &b
		return nil
	}
	if value.Tag == "!!str" {
		c.String = &value.Value
		return nil
	}
	return fmt.Errorf("ContextConfig: cannot unmarshal YAML tag %q", value.Tag)
}
