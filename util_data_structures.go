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
	"reflect"
)

// ProgMap is a custom map type that allows for incremental unmarshaling of JSON data.
// Instead of replacing the entire map contents during unmarshaling, it merges new key-value
// pairs into the existing map, preserving any entries that aren't overwritten by the incoming JSON.
type ProgMap map[string]any

// UnmarshalJSON implements json.Unmarshaler to merge incoming JSON data into the existing map
// rather than replacing it entirely. This allows for progressive/incremental updates where
// new fields are added without losing existing fields not present in the incoming JSON.
func (p *ProgMap) UnmarshalJSON(data []byte) error {
	newData := make(map[string]any)
	if err := json.Unmarshal(data, &newData); err != nil {
		return err
	}
	for k, v := range newData {
		(*p)[k] = v
	}
	return nil
}

// emptyMap is a map[string]any that's initialized with no entries.
var emptyMap = make(map[string]any)

// isMapAny returns true if the given data is a map[string]any
func isMapAny(data any) bool {
	_, ok := data.(map[string]any)
	return ok
}

// AnyToResultMap converts any data into a map[string]any. if the data is already a map[string]any, it's returned as-is,
// otherwise a map with a single entry with key "result" is returned.
func AnyToResultMap(data any) map[string]any {
	switch t := data.(type) {
	case map[string]any:
		return t
	default:
		return map[string]any{"result": data}
	}
}

// initDataStructure initializes the data structure, assuming it's either a map or a struct
func initDataStructure[T any]() *T {
	var v T
	val := reflect.ValueOf(&v).Elem()
	if val.Kind() == reflect.Map {
		val.Set(reflect.MakeMap(val.Type()))
		return &v
	} else {
		return new(T)
	}
}
