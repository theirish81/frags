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

func TestTransformer_Transform(t *testing.T) {
	tx := Transformer{
		Name:    "foo",
		Jsonata: strPtr(`result.{"first_name":first_name,"last_name":last_name}`),
	}
	res, err := tx.Transform(map[string]any{
		"result": map[string]any{"first_name": "John", "last_name": "Doe", "address": "123 Main St"},
	})
	assert.Nil(t, err)
	assert.Equal(t, map[string]any{"first_name": "John", "last_name": "Doe"}, res)

	tx = Transformer{
		Name:    "foo",
		Jsonata: strPtr(`{ "result": {"first_name":result.first_name, "last_name":result.last_name }}`),
	}
	res, err = tx.Transform(map[string]any{
		"result": map[string]any{"first_name": "John", "last_name": "Doe", "address": "123 Main St"},
	})
	assert.Nil(t, err)
	assert.Equal(t, map[string]any{"result": map[string]any{"first_name": "John", "last_name": "Doe"}}, res)

	tx = Transformer{
		Name:    "foo",
		Jsonata: strPtr(`{ "result": [result.{"first_name":first_name, "last_name":last_name }]}`),
	}
	res, err = tx.Transform(map[string]any{
		"result": []map[string]any{
			{"first_name": "John", "last_name": "Doe", "address": "123 Main St"},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, map[string]any{"result": []any{map[string]any{"first_name": "John", "last_name": "Doe"}}}, res)
}
