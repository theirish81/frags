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
	"log/slog"
	"testing"
	"time"

	"github.com/blues/jsonata-go"
	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/assert"
)

func TestTransformer_Transform(t *testing.T) {
	t.Run("JSONATA transform a result map", func(t *testing.T) {
		tx := Transformer{
			Name:    "foo",
			Jsonata: StrPtr(`result.{"first_name":first_name,"last_name":last_name}`),
		}
		res, err := tx.Transform(NewFragsContext(time.Minute), map[string]any{
			"result": map[string]any{"first_name": "John", "last_name": "Doe", "address": "123 Main St"},
		}, &Runner[any]{logger: NewStreamerLogger(slog.Default(), nil, DebugChannelLevel)})
		assert.Nil(t, err)
		assert.Equal(t, map[string]any{"first_name": "John", "last_name": "Doe"}, res)
	})
	t.Run("JSONATA transform a map, alternative syntax", func(t *testing.T) {
		tx := Transformer{
			Name:    "foo",
			Jsonata: StrPtr(`{ "result": {"first_name":result.first_name, "last_name":result.last_name }}`),
		}
		res, err := tx.Transform(NewFragsContext(time.Minute), map[string]any{
			"result": map[string]any{"first_name": "John", "last_name": "Doe", "address": "123 Main St"},
		}, &Runner[any]{logger: NewStreamerLogger(slog.Default(), nil, DebugChannelLevel)})
		assert.Nil(t, err)
		assert.Equal(t, map[string]any{"result": map[string]any{"first_name": "John", "last_name": "Doe"}}, res)
	})
	t.Run("JSONATA transform a result map containing an array", func(t *testing.T) {
		tx := Transformer{
			Name:    "foo",
			Jsonata: StrPtr(`{ "result": [result.{"first_name":first_name, "last_name":last_name }]}`),
		}
		res, err := tx.Transform(NewFragsContext(time.Minute), map[string]any{
			"result": []map[string]any{
				{"first_name": "John", "last_name": "Doe", "address": "123 Main St"},
			},
		}, &Runner[any]{logger: NewStreamerLogger(slog.Default(), nil, DebugChannelLevel)})
		assert.Nil(t, err)
		assert.Equal(t, AnyToResultMap([]any{map[string]any{"first_name": "John", "last_name": "Doe"}}), res)
	})
	t.Run("JSONATA transform a regular map", func(t *testing.T) {
		tx := Transformer{
			Name:    "foo",
			Jsonata: StrPtr(`{"first_name":first_name,"last_name":last_name}`),
		}
		res, err := tx.Transform(NewFragsContext(time.Minute), map[string]any{"first_name": "John", "last_name": "Doe", "address": "123 Main St"},
			&Runner[any]{logger: NewStreamerLogger(slog.Default(), nil, DebugChannelLevel)})
		assert.Nil(t, err)
		assert.Equal(t, map[string]any{"first_name": "John", "last_name": "Doe"}, res)
	})
	t.Run("JSONATA transform a regular array", func(t *testing.T) {
		tx := Transformer{
			Name:    "foo",
			Jsonata: StrPtr(`[{"first_name":first_name, "last_name":last_name }]`),
		}
		res, err := tx.Transform(NewFragsContext(time.Minute), []map[string]any{
			{"first_name": "John", "last_name": "Doe", "address": "123 Main St"},
		}, &Runner[any]{logger: NewStreamerLogger(slog.Default(), nil, DebugChannelLevel)})
		assert.Nil(t, err)
		assert.Equal(t, []any{map[string]any{"first_name": "John", "last_name": "Doe"}}, res)
	})
	t.Run("JSON Parser+JSONATA transform a regular map", func(t *testing.T) {
		px := JsonParser
		tx := Transformer{
			Name:    "foo",
			Jsonata: StrPtr(`{"first_name":first_name,"last_name":last_name}`),
			Parser:  &px,
		}
		res, err := tx.Transform(NewFragsContext(time.Minute), `{"first_name": "John", "last_name": "Doe", "address": "123 Main St"}`,
			&Runner[any]{logger: NewStreamerLogger(slog.Default(), nil, DebugChannelLevel)})
		assert.Nil(t, err)
		assert.Equal(t, map[string]any{"first_name": "John", "last_name": "Doe"}, res)
	})
	t.Run("JSON Parser+JSONATA transform a regular array", func(t *testing.T) {
		px := JsonParser
		tx := Transformer{
			Name:    "foo",
			Jsonata: StrPtr(`[{"first_name":first_name, "last_name":last_name }]`),
			Parser:  &px,
		}
		res, err := tx.Transform(NewFragsContext(time.Minute), `[{"first_name": "John", "last_name": "Doe", "address": "123 Main St"}]`,
			&Runner[any]{logger: NewStreamerLogger(slog.Default(), nil, DebugChannelLevel)})
		assert.Nil(t, err)
		assert.Equal(t, []any{map[string]any{"first_name": "John", "last_name": "Doe"}}, res)
	})
	t.Run("CSV Parser+JSONATA transform a regular array", func(t *testing.T) {
		px := CsvParser
		tx := Transformer{
			Name:    "foo",
			Jsonata: StrPtr(`$[].{"first_name":$[0], "last_name":$[1] }`),
			Parser:  &px,
		}
		res, err := tx.Transform(NewFragsContext(time.Minute), `John,Doe`, &Runner[any]{logger: NewStreamerLogger(slog.Default(), nil, DebugChannelLevel)})
		assert.Nil(t, err)
		assert.Equal(t, []any{map[string]any{"first_name": "John", "last_name": "Doe"}}, res)
	})

}

func TestTransformer_Transform2(t *testing.T) {
	data := s1{
		S2: s2{
			P1: 1,
			P2: 2,
		},
	}
	script, err := jsonata.Compile(`
{
  	"s2": {
		"p1": S2.P1 * 2
		}
}
`)
	assert.Nil(t, err)
	res, err := script.Eval(data)
	assert.Nil(t, err)
	sx := s1{}
	dec, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &sx,
	})
	assert.Nil(t, dec.Decode(res))
	assert.Equal(t, float64(2), sx.S2.P1)
}
