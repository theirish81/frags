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
	"fmt"
	"reflect"
	"sort"
)

// GuessSchema inspects a Go value (map, slice, scalar) and returns a JSON Schema.
// Supported input types: map[string]any, []any, primitives, nil.
func GuessSchema(v any) *Schema {
	return guessValue(reflect.ValueOf(v))
}

func guessValue(rv reflect.Value) *Schema {
	// Unwrap interfaces and pointers
	for rv.IsValid() && (rv.Kind() == reflect.Interface || rv.Kind() == reflect.Ptr) {
		if rv.IsNil() {
			return nullable()
		}
		rv = rv.Elem()
	}

	if !rv.IsValid() {
		return nullable()
	}

	switch rv.Kind() {
	case reflect.Map:
		return guessMap(rv)
	case reflect.Slice, reflect.Array:
		return guessSlice(rv)
	case reflect.String:
		return &Schema{Type: String}
	case reflect.Bool:
		return &Schema{Type: Boolean}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: Integer}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: Integer}
	case reflect.Float32, reflect.Float64:
		return guessNumber(rv)
	default:
		return &Schema{Type: String} // fallback
	}
}

func guessMap(rv reflect.Value) *Schema {
	s := &Schema{
		Type:       Object,
		Properties: make(map[string]*Schema),
	}

	for _, key := range rv.MapKeys() {
		name := fmt.Sprintf("%v", key.Interface())
		val := rv.MapIndex(key)
		s.Properties[name] = guessValue(val)
	}

	// Stable property ordering
	s.PropertyOrdering = sortedKeys(s.Properties)

	return s
}

func guessSlice(rv reflect.Value) *Schema {
	s := &Schema{Type: Array}

	n := rv.Len()
	if n == 0 {
		return s // no items to inspect
	}

	// Collect one schema per element
	schemas := make([]*Schema, n)
	for i := 0; i < n; i++ {
		schemas[i] = guessValue(rv.Index(i))
	}

	// Merge all element schemas into one representative items schema
	s.Items = mergeSchemas(schemas)
	return s
}

func mergeSchemas(schemas []*Schema) *Schema {
	if len(schemas) == 0 {
		return &Schema{}
	}
	if len(schemas) == 1 {
		return schemas[0]
	}

	// Group by type
	groups := groupByType(schemas)

	// If only one type is present, merge within that type
	if len(groups) == 1 {
		for _, group := range groups {
			return mergeSameType(group)
		}
	}

	// Multiple types → oneOf, but first try to merge within each type group
	var branches []*Schema
	for _, group := range sortedGroups(groups) {
		branches = append(branches, mergeSchemas(group))
	}

	// Simplify: if one branch is nullable and the other is not, fold into nullable
	if simplified, ok := trySimplifyNullable(branches); ok {
		return simplified
	}

	return &Schema{OneOf: branches}
}

// mergeSchemas for schemas that share the same type
func mergeSameType(schemas []*Schema) *Schema {
	if len(schemas) == 0 {
		return &Schema{}
	}

	base := schemas[0]

	switch base.Type {
	case Object:
		return mergeObjects(schemas)
	case Array:
		return mergeArrays(schemas)
	default:
		// scalar: just return the type, no merging needed
		return &Schema{Type: base.Type}
	}
}

func mergeObjects(schemas []*Schema) *Schema {
	merged := &Schema{
		Type:       Object,
		Properties: make(map[string]*Schema),
	}

	// property name → list of schemas seen for that property
	propSchemas := make(map[string][]*Schema)
	// property name → count of objects that have this property
	propCount := make(map[string]int)

	for _, s := range schemas {
		for name, propSchema := range s.Properties {
			propSchemas[name] = append(propSchemas[name], propSchema)
			propCount[name]++
		}
	}

	// Merge each property
	for name, pSchemas := range propSchemas {
		merged.Properties[name] = mergeSchemas(pSchemas)
	}

	// Required = properties present in ALL objects
	total := len(schemas)
	var required []string
	for name, count := range propCount {
		if count == total {
			required = append(required, name)
		}
	}
	sort.Strings(required)
	if len(required) > 0 {
		merged.Required = required
	}

	merged.PropertyOrdering = sortedKeys(merged.Properties)
	return merged
}

func mergeArrays(schemas []*Schema) *Schema {
	var itemSchemas []*Schema
	for _, s := range schemas {
		if s.Items != nil {
			itemSchemas = append(itemSchemas, s.Items)
		}
	}
	merged := &Schema{Type: Array}
	if len(itemSchemas) > 0 {
		merged.Items = mergeSchemas(itemSchemas)
	}
	return merged
}

func nullable() *Schema {
	t := true
	return &Schema{Nullable: &t}
}

// guessNumber: returns "integer" if the float has no fractional part, else "number"
func guessNumber(rv reflect.Value) *Schema {
	f := rv.Float()
	if f == float64(int64(f)) {
		return &Schema{Type: Integer}
	}
	return &Schema{Type: Number}
}

// groupByType groups schemas by their Type field.
// Schemas with no Type (e.g. nullable) are placed under "null".
func groupByType(schemas []*Schema) map[string][]*Schema {
	groups := make(map[string][]*Schema)
	for _, s := range schemas {
		t := s.Type
		if t == "" {
			if s.Nullable != nil && *s.Nullable {
				t = "null"
			} else if len(s.OneOf) > 0 {
				t = "oneOf"
			} else {
				t = "unknown"
			}
		}
		groups[string(t)] = append(groups[string(t)], s)
	}
	return groups
}

// sortedGroups returns group slices in a stable key order
func sortedGroups(groups map[string][]*Schema) [][]*Schema {
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([][]*Schema, 0, len(keys))
	for _, k := range keys {
		result = append(result, groups[k])
	}
	return result
}

// trySimplifyNullable: if branches = [nullable, X], return X with Nullable=true
func trySimplifyNullable(branches []*Schema) (*Schema, bool) {
	if len(branches) != 2 {
		return nil, false
	}
	nullIdx := -1
	otherIdx := -1
	for i, b := range branches {
		if b.Type == "" && b.Nullable != nil && *b.Nullable {
			nullIdx = i
		} else {
			otherIdx = i
		}
	}
	if nullIdx == -1 || otherIdx == -1 {
		return nil, false
	}
	result := *branches[otherIdx] // shallow copy
	t := true
	result.Nullable = &t
	return &result, true
}

func sortedKeys(m map[string]*Schema) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
