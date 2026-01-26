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
	"testing"

	"github.com/stretchr/testify/assert"
)

type s2 struct {
	P1 float64 `json:"p1"`
	P2 float64 `json:"p2"`
}

type s3 struct {
	M1 float64 `json:"m1"`
	M2 float64 `json:"m2"`
}

type s1 struct {
	S2        s2            `json:"s2"`
	ArrayOfS2 []s2          `json:"arrayOfS2"`
	MapOfS3   map[string]s3 `json:"mapOfS3"`
}

func TestSchema_Validate(t *testing.T) {
	t.Run("base type", func(t *testing.T) {
		s := Schema{Type: SchemaString}
		err := s.Validate("foo", nil)
		assert.NoError(t, err)

		s = Schema{Type: SchemaInteger}
		err = s.Validate("foo", nil)
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
		err := s.Validate(map[string]any{"foo": "123", "bar": 123}, nil)
		assert.NoError(t, err)

		err = s.Validate(map[string]any{"foo": "123"}, nil)
		assert.NoError(t, err)

		err = s.Validate(map[string]any{"foo": 123}, nil)
		assert.Error(t, err)

		s.Required = []string{"foo", "bar"}
		err = s.Validate(map[string]any{"foo": "123"}, nil)
		assert.Error(t, err)
	})
	t.Run("array", func(t *testing.T) {
		s := Schema{
			Type:  SchemaArray,
			Items: &Schema{Type: SchemaString},
		}
		err := s.Validate([]string{"foo", "bar"}, nil)
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
		err = s.Validate([]map[string]any{{"foo": "123", "bar": 123}}, nil)
		assert.NoError(t, err)

		err = s.Validate([]map[string]any{{"foo": "123", "bar": 123}, {"foo": "123", "bar": "123"}}, nil)
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
		err := s.Validate(struct1, nil)
		assert.NoError(t, err)
	})
}

func TestSchema_Validate_Soft(t *testing.T) {
	t.Run("all matches", func(t *testing.T) {
		s := Schema{
			Type: SchemaObject,
			Properties: map[string]*Schema{
				"int":     {Type: SchemaInteger},
				"str":     {Type: SchemaString},
				"bool":    {Type: SchemaBoolean},
				"realInt": {Type: SchemaInteger},
			},
		}
		err := s.Validate(map[string]any{
			"int":     "123",
			"str":     "123",
			"bool":    "true",
			"realInt": 123,
		}, &ValidatorOptions{SoftValidation: true})
		assert.NoError(t, err)
	})
	t.Run("unmatched promise", func(t *testing.T) {
		s := Schema{
			Type: SchemaObject,
			Properties: map[string]*Schema{
				"int":  {Type: SchemaInteger},
				"str":  {Type: SchemaString},
				"bool": {Type: SchemaBoolean},
			},
		}
		err := s.Validate(map[string]any{
			"int":  "abc",
			"str":  "123",
			"bool": true,
		}, &ValidatorOptions{SoftValidation: true})
		assert.Error(t, err)
	})
	t.Run("unmatched promise", func(t *testing.T) {
		s := Schema{
			Type: SchemaObject,
			Properties: map[string]*Schema{
				"int":  {Type: SchemaInteger},
				"str":  {Type: SchemaString},
				"bool": {Type: SchemaBoolean},
			},
		}
		err := s.Validate(map[string]any{
			"int":  "123",
			"str":  "123",
			"bool": "foo",
		}, &ValidatorOptions{SoftValidation: true})
		assert.Error(t, err)
	})

}
