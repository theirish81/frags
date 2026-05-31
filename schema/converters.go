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

	"github.com/jinzhu/copier"
)

// UnmarshalMapstructure is necessary here because of type behaving both as a string or array of strings
func (s *Schema) UnmarshalMapstructure(data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("schema: mapstructure marshal: %w", err)
	}
	return json.Unmarshal(b, s)
}

func CopyConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		{
			SrcType: []any{},
			DstType: []string{},
			Fn: func(src any) (any, error) {
				s := src.([]any)
				res := make([]string, len(s))
				for i, v := range s {
					res[i] = fmt.Sprint(v)
				}
				return res, nil
			},
		},
		{
			SrcType: []any{},
			DstType: Type(""),
			Fn: func(src any) (any, error) {
				arr, ok := src.([]any)
				if !ok {
					return Type(""), fmt.Errorf("schema type: expected []any, got %T", src)
				}
				for _, item := range arr {
					if s, ok := item.(string); ok && s != "null" {
						return Type(s), nil
					}
				}
				return Type(""), nil
			},
		},
		{
			SrcType: []string{},
			DstType: Type(""),
			Fn: func(src any) (any, error) {
				arr, ok := src.([]string)
				if !ok {
					return Type(""), fmt.Errorf("schema type: expected []any, got %T", src)
				}
				for _, item := range arr {
					if item != "null" {
						return Type(item), nil
					}
				}
				return Type(""), nil
			},
		},
	}
}

// CopyFrom copies src into this Schema with the correct type converters.
func (s *Schema) CopyFrom(src any) error {
	return copier.CopyWithOption(s, src, copier.Option{
		Converters: CopyConverters(),
	})
}

// CopyTo copies this Schema into dst with the correct type converters.
func (s *Schema) CopyTo(dst any) error {
	return copier.CopyWithOption(dst, s, copier.Option{
		Converters: CopyConverters(),
	})
}
