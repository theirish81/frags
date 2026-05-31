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
	"testing"

	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/theirish81/frags/util"
)

func TestFromAny_MapstructureTypeArray(t *testing.T) {
	data := map[string]any{
		"type": []any{"object", "null"},
		"properties": map[string]any{
			"name": map[string]any{
				"type": []any{"string", "null"},
			},
			"age": map[string]any{
				"type": "integer",
			},
		},
	}

	s, err := FromAny(data)
	assert.NoError(t, err)

	assert.Equal(t, Type("object"), s.Type)
	assert.Equal(t, util.Ptr(true), s.Nullable)

	assert.Equal(t, Type("string"), s.Properties["name"].Type)
	assert.Equal(t, util.Ptr(true), s.Properties["name"].Nullable)

	assert.Equal(t, Type("integer"), s.Properties["age"].Type)
	assert.Nil(t, s.Properties["age"].Nullable)
}

func TestFromAny_CopierTypeConverter(t *testing.T) {
	type SourceSchema struct {
		Type       []string
		Nullable   *bool
		Properties map[string]SourceSchema
	}

	src := SourceSchema{
		Type: []string{"object", "null"},
		Properties: map[string]SourceSchema{
			"name": {Type: []string{"string", "null"}},
			"age":  {Type: []string{"integer"}},
		},
	}

	dst := &Schema{}
	err := copier.CopyWithOption(dst, src, copier.Option{
		Converters: CopyConverters(),
	})

	assert.NoError(t, err)
	assert.Equal(t, Type("object"), dst.Type)
	assert.Equal(t, Type("string"), dst.Properties["name"].Type)
	assert.Equal(t, Type("integer"), dst.Properties["age"].Type)
}

func TestCopyConverter_StringToType(t *testing.T) {
	type SourceSchema struct {
		Type string
	}

	src := SourceSchema{Type: "object"}
	dst := &Schema{}

	err := copier.CopyWithOption(dst, src, copier.Option{
		Converters: CopyConverters(),
	})

	assert.NoError(t, err)
	assert.Equal(t, Type("object"), dst.Type)
}
