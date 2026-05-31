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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/theirish81/frags/util"
	"gopkg.in/yaml.v3"
)

func TestType_PlainString_JSON(t *testing.T) {
	s := &Schema{}
	err := json.Unmarshal([]byte(`{"type":"string"}`), s)
	assert.NoError(t, err)
	assert.Equal(t, Type("string"), s.Type)
	assert.Nil(t, s.Nullable)
}

func TestType_PlainString_YAML(t *testing.T) {
	s := &Schema{}
	err := yaml.Unmarshal([]byte("type: string\n"), s)
	assert.NoError(t, err)
	assert.Equal(t, Type("string"), s.Type)
	assert.Nil(t, s.Nullable)
}

func TestType_ArrayNullLast_JSON(t *testing.T) {
	s := &Schema{}
	err := json.Unmarshal([]byte(`{"type":["string","null"]}`), s)
	assert.NoError(t, err)
	assert.Equal(t, Type("string"), s.Type)
	assert.Equal(t, util.Ptr(true), s.Nullable)
}

func TestType_ArrayNullLast_YAML(t *testing.T) {
	s := &Schema{}
	err := yaml.Unmarshal([]byte("type:\n  - string\n  - null\n"), s)
	assert.NoError(t, err)
	assert.Equal(t, Type("string"), s.Type)
	assert.Equal(t, util.Ptr(true), s.Nullable)
}

func TestType_ArrayNullFirst_JSON(t *testing.T) {
	s := &Schema{}
	err := json.Unmarshal([]byte(`{"type":["null","integer"]}`), s)
	assert.NoError(t, err)
	assert.Equal(t, Type("integer"), s.Type)
	assert.Equal(t, util.Ptr(true), s.Nullable)
}

func TestType_ArrayNullFirst_YAML(t *testing.T) {
	s := &Schema{}
	err := yaml.Unmarshal([]byte("type:\n  - null\n  - integer\n"), s)
	assert.NoError(t, err)
	assert.Equal(t, Type("integer"), s.Type)
	assert.Equal(t, util.Ptr(true), s.Nullable)
}

func TestType_ArrayOnlyNull_JSON(t *testing.T) {
	s := &Schema{}
	err := json.Unmarshal([]byte(`{"type":["null"]}`), s)
	assert.NoError(t, err)
	assert.Equal(t, Type(""), s.Type)
	assert.Equal(t, util.Ptr(true), s.Nullable)
}

func TestType_ArrayOnlyNull_YAML(t *testing.T) {
	s := &Schema{}
	err := yaml.Unmarshal([]byte("type:\n  - null\n"), s)
	assert.NoError(t, err)
	assert.Equal(t, Type(""), s.Type)
	assert.Equal(t, util.Ptr(true), s.Nullable)
}

func TestType_ArrayPicksFirstNonNull_JSON(t *testing.T) {
	// when multiple non-null types are present, the first one wins
	s := &Schema{}
	err := json.Unmarshal([]byte(`{"type":["object","string","null"]}`), s)
	assert.NoError(t, err)
	assert.Equal(t, Type("object"), s.Type)
	assert.Equal(t, util.Ptr(true), s.Nullable)
}

func TestType_ArrayPicksFirstNonNull_YAML(t *testing.T) {
	s := &Schema{}
	err := yaml.Unmarshal([]byte("type:\n  - object\n  - string\n  - null\n"), s)
	assert.NoError(t, err)
	assert.Equal(t, Type("object"), s.Type)
	assert.Equal(t, util.Ptr(true), s.Nullable)
}

func TestType_NestedPropertyArray_JSON(t *testing.T) {
	s := &Schema{}
	err := json.Unmarshal([]byte(`{
		"type": "object",
		"properties": {
			"name": {"type": ["string","null"]},
			"age":  {"type": "integer"}
		}
	}`), s)
	assert.NoError(t, err)
	assert.Equal(t, Type("string"), s.Properties["name"].Type)
	assert.Equal(t, util.Ptr(true), s.Properties["name"].Nullable)
	assert.Equal(t, Type("integer"), s.Properties["age"].Type)
	assert.Nil(t, s.Properties["age"].Nullable)
}

func TestType_NestedPropertyArray_YAML(t *testing.T) {
	s := &Schema{}
	err := yaml.Unmarshal([]byte(`
type: object
properties:
  name:
    type:
      - string
      - null
  age:
    type: integer
`), s)
	assert.NoError(t, err)
	assert.Equal(t, Type("string"), s.Properties["name"].Type)
	assert.Equal(t, util.Ptr(true), s.Properties["name"].Nullable)
	assert.Equal(t, Type("integer"), s.Properties["age"].Type)
	assert.Nil(t, s.Properties["age"].Nullable)
}

func TestType_MarshalAlwaysEmitsString_JSON(t *testing.T) {
	s := &Schema{}
	err := json.Unmarshal([]byte(`{"type":["string","null"]}`), s)
	assert.NoError(t, err)

	out, err := json.Marshal(s)
	assert.NoError(t, err)

	s2 := &Schema{}
	err = json.Unmarshal(out, s2)
	assert.NoError(t, err)
	assert.Equal(t, Type("string"), s2.Type)
}

func TestType_MarshalAlwaysEmitsString_YAML(t *testing.T) {
	s := &Schema{}
	err := yaml.Unmarshal([]byte("type:\n  - string\n  - null\n"), s)
	assert.NoError(t, err)

	out, err := yaml.Marshal(s)
	assert.NoError(t, err)

	s2 := &Schema{}
	err = yaml.Unmarshal(out, s2)
	assert.NoError(t, err)
	assert.Equal(t, Type("string"), s2.Type)
}
