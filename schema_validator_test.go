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

package frags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchema_Validate(t *testing.T) {
	t.Run("base type", func(t *testing.T) {
		s := Schema{Type: SchemaString}
		err := s.Validate("foo")
		assert.NoError(t, err)

		s = Schema{Type: SchemaInteger}
		err = s.Validate("foo")
		assert.Error(t, err)
	})
	t.Run("map", func(t *testing.T) {
		s := Schema{
			Type: SchemaObject,
			Properties: map[string]*Schema{
				"foo": {Type: SchemaString},
				"bar": {Type: SchemaInteger},
			},
		}
		err := s.Validate(map[string]any{"foo": "123", "bar": 123})
		assert.NoError(t, err)

		err = s.Validate(map[string]any{"foo": "123"})
		assert.NoError(t, err)

		err = s.Validate(map[string]any{"foo": 123})
		assert.Error(t, err)

		s.Required = []string{"foo", "bar"}
		err = s.Validate(map[string]any{"foo": "123"})
		assert.Error(t, err)
	})
	t.Run("array", func(t *testing.T) {
		s := Schema{
			Type:  SchemaArray,
			Items: &Schema{Type: SchemaString},
		}
		err := s.Validate([]string{"foo", "bar"})
		assert.NoError(t, err)

		s = Schema{
			Type: SchemaArray,
			Items: &Schema{
				Type:     SchemaObject,
				Required: []string{"foo", "bar"},
				Properties: map[string]*Schema{
					"foo": {Type: SchemaString},
					"bar": {Type: SchemaInteger},
				},
			},
		}
		err = s.Validate([]map[string]any{{"foo": "123", "bar": 123}})
		assert.NoError(t, err)

		err = s.Validate([]map[string]any{{"foo": "123", "bar": 123}, {"foo": "123", "bar": "123"}})
		assert.Error(t, err)
	})
	t.Run("composite struct", func(t *testing.T) {
		s := Schema{
			Type:     SchemaObject,
			Required: []string{"s2", "arrayOfS2", "mapOfS3"},
			Properties: map[string]*Schema{
				"s2": {
					Type:     SchemaObject,
					Required: []string{"p1", "p2"},
					Properties: map[string]*Schema{
						"p1": {Type: SchemaNumber},
						"p2": {Type: SchemaInteger},
					},
				},
				"arrayOfS2": {
					Type: SchemaArray,
					Items: &Schema{
						Type: SchemaObject,
						Properties: map[string]*Schema{
							"p1": {Type: SchemaNumber},
							"p2": {Type: SchemaInteger},
						},
					},
				},
				"mapOfS3": {
					Type:     SchemaObject,
					Required: []string{"v1"},
					Properties: map[string]*Schema{
						"v1": {
							Type:     SchemaObject,
							Required: []string{"m1", "m2"},
							Properties: map[string]*Schema{
								"m1": {Type: SchemaInteger},
								"m2": {Type: SchemaInteger},
							},
						},
					},
				},
			},
		}
		struct1 := s1{
			S2: s2{
				P1: 32.5,
				P2: 123,
			},
			ArrayOfS2: []s2{{
				P1: 32.5,
				P2: 123,
			}},
			MapOfS3: map[string]s3{
				"v1": {
					M1: 123,
					M2: 456,
				},
			},
		}
		err := s.Validate(struct1)
		assert.NoError(t, err)
	})
}
