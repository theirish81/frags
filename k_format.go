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
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ToKFormat renders a value in a format we called "K Format" or "Knowledge Format", a flavor of markdown that is
// designed to be easily readable by LLMs.
func ToKFormat(v any) string {
	return recursiveRender(reflect.ValueOf(v), 0)
}

func recursiveRender(val reflect.Value, depth int) string {
	if !val.IsValid() {
		// Ideally this shouldn't happen, but just in case
		return "`<NULL>`"
	}

	// Dereference pointers/interfaces
	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return fmt.Sprintf("`<NULL: %s>`", val.Type().String())
		}
		return recursiveRender(val.Elem(), depth)
	}

	// Space indentation for nested structures
	indent := strings.Repeat("  ", depth)
	var sb strings.Builder

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		if val.Len() == 0 {
			return " (empty)"
		}
		// Byte slices are often binary data; denote them explicitly. This is probably pointless, I have serious
		// doubts about the LLM's ability to understand binary data.
		if val.Type().Elem().Kind() == reflect.Uint8 {
			return fmt.Sprintf("`(bytes)` %v", val.Bytes())
		}

		sb.WriteString("\n")
		for i := 0; i < val.Len(); i++ {
			element := val.Index(i)
			// We include the index for the LLM's spatial awareness
			sb.WriteString(fmt.Sprintf("%s- [%d] %s\n", indent, i, recursiveRender(element, depth+1)))
		}
		return strings.TrimRight(sb.String(), "\n")

	case reflect.Map:
		if val.Len() == 0 {
			return "(empty)"
		}
		sb.WriteString("\n")

		// 1. Collect keys
		keys := val.MapKeys()

		// 2. Sort keys deterministically (String representation sort)
		// This is CRITICAL for LLMs to see consistent patterns.
		sort.Slice(keys, func(i, j int) bool {
			return fmt.Sprintf("%v", keys[i]) < fmt.Sprintf("%v", keys[j])
		})

		// 3. Iterate sorted keys
		for _, k := range keys {
			v := val.MapIndex(k)
			keyStr := fmt.Sprintf("%v", k)
			if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
				continue
			}
			sb.WriteString(fmt.Sprintf("%s- **%s**: %s\n", indent, keyStr, recursiveRender(v, depth+1)))
		}
		return strings.TrimRight(sb.String(), "\n")

	case reflect.Struct:
		sb.WriteString("\n")
		t := val.Type()
		// Struct fields are already ordered by definition, but we filter unexported
		for i := 0; i < val.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue // unexported
			}
			fieldValue := val.Field(i)
			sb.WriteString(fmt.Sprintf("%s- **%s**: %s\n", indent, field.Name, recursiveRender(fieldValue, depth+1)))
		}
		return strings.TrimRight(sb.String(), "\n")

	case reflect.String:
		return fmt.Sprintf("`(string)` \"%s\"", val.String())

	case reflect.Bool:
		return fmt.Sprintf("`(bool)` %v", val.Bool())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("`(int)` %d", val.Int())

	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("`(float)` %v", val.Float())

	default:
		// Fallback for complex numbers, channels, funcs etc. These most likely will be just noise, but the
		// expectation is the user will provide clear objects for this purpose...
		return fmt.Sprintf("`(%s)` %v", val.Kind().String(), val)
	}
}
