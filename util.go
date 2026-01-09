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
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

var emptyMap = make(map[string]any)

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

// parseDurationOrDefault parses a duration string into a time.Duration, or returns the default duration if parsing fails
func parseDurationOrDefault(durationStr *string, defaultDuration time.Duration) time.Duration {
	if durationStr == nil || *durationStr == "" {
		return defaultDuration
	}
	parsedDuration, err := time.ParseDuration(*durationStr)
	if err != nil {
		return defaultDuration
	}
	return parsedDuration
}

func strPtr(s string) *string { return &s }

// ConvertToMapAny converts a map[string]S to a map[string]any
func ConvertToMapAny[S any](source map[string]S) map[string]any {
	t := make(map[string]any)
	for k, v := range source {
		t[k] = v
	}
	return t
}

func anyToResultMap(data any) map[string]any {
	switch t := data.(type) {
	case map[string]any:
		return t
	default:
		return map[string]any{"result": data}
	}
}

func parseJSON(data any) (any, error) {
	switch t := data.(type) {
	case map[string]any:
		return t, nil
	case []any:
		return t, nil
	case []byte:
		mapOut := make(map[string]any)
		if err := json.Unmarshal([]byte(t), &mapOut); err == nil {
			return mapOut, nil
		}
		sliceOut := make([]any, 0)
		if err := json.Unmarshal([]byte(t), &sliceOut); err == nil {
			return sliceOut, nil
		}
	case string:
		return parseJSON([]byte(t))
	}
	return nil, errors.New("cannot parse this from anything into anything")
}

func parseCSV(data any) ([][]string, error) {
	switch t := data.(type) {
	case string:
		reader := csv.NewReader(strings.NewReader(t))
		return reader.ReadAll()
	case []byte:
		reader := csv.NewReader(bytes.NewReader(t))
		return reader.ReadAll()
	}
	return nil, errors.New("cannot parse this from CSV into anything")
}

func replaceExtension(filename, newExt string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return filename + newExt
	}
	return filename[:len(filename)-len(ext)] + newExt
}
