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
	"reflect"
	"strings"
)

func StructToSchema(v any) *Schema {
	t := reflect.TypeOf(v)

	if v == nil {
		return nil
	}
	// Handle pointer types
	if t.Kind() == reflect.Ptr {

		t = t.Elem()
	}

	// Only process structs
	if t.Kind() != reflect.Struct {
		return nil
	}

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Required:   []string{},
	}

	// Process each field in the struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag name, default to field name
		jsonTag := field.Tag.Get("json")
		fieldName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
			// Skip if json:"-"
			if jsonTag == "-" {
				continue
			}
		}

		// Create field schema
		fieldSchema := createFieldSchema(field)

		// Parse frags tag
		fragsTag := field.Tag.Get("frags")
		if fragsTag != "" {
			parseFragsTag(fieldSchema, fragsTag)
		}

		schema.Properties[fieldName] = fieldSchema
		schema.Required = append(schema.Required, fieldName)
	}

	return schema
}

// createFieldSchema creates a Schema for a single field
func createFieldSchema(field reflect.StructField) *Schema {
	schema := &Schema{}

	fieldType := field.Type

	// Handle pointer types
	if fieldType.Kind() == reflect.Ptr {
		nullable := true
		schema.Nullable = &nullable
		fieldType = fieldType.Elem()
	}

	switch fieldType.Kind() {
	case reflect.String:
		schema.Type = "string"

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"

	case reflect.Float32, reflect.Float64:
		schema.Type = "number"

	case reflect.Bool:
		schema.Type = "boolean"

	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		// Create schema for array items
		itemType := fieldType.Elem()
		schema.Items = &Schema{}

		if itemType.Kind() == reflect.Ptr {
			itemType = itemType.Elem()
		}

		switch itemType.Kind() {
		case reflect.String:
			schema.Items.Type = "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			schema.Items.Type = "integer"
		case reflect.Float32, reflect.Float64:
			schema.Items.Type = "number"
		case reflect.Bool:
			schema.Items.Type = "boolean"
		case reflect.Struct:
			// Recursively process struct items
			schema.Items = StructToSchema(reflect.New(itemType).Elem().Interface())
		default:
			schema.Items.Type = "object"
		}

	case reflect.Map:
		schema.Type = "object"

	case reflect.Struct:
		// Recursively process nested structs
		nestedSchema := StructToSchema(reflect.New(fieldType).Elem().Interface())
		if nestedSchema != nil {
			*schema = *nestedSchema
		}

	default:
		schema.Type = "object"
	}

	return schema
}

// parseFragsTag parses the frags struct tag and populates the schema
func parseFragsTag(schema *Schema, tag string) {
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.HasPrefix(part, "x-session=") {
			value := strings.TrimPrefix(part, "x-session=")
			schema.XSession = &value
		} else if strings.HasPrefix(part, "description=") {
			schema.Description = strings.TrimPrefix(part, "description=")
		} else if strings.HasPrefix(part, "enum=") {
			enumStr := strings.TrimPrefix(part, "enum=")
			schema.Enum = strings.Split(enumStr, "|")
		} else if strings.HasPrefix(part, "format=") {
			schema.Format = strings.TrimPrefix(part, "format=")
		} else if strings.HasPrefix(part, "pattern=") {
			schema.Pattern = strings.TrimPrefix(part, "pattern=")
		} else if strings.HasPrefix(part, "title=") {
			schema.Title = strings.TrimPrefix(part, "title=")
		} else if strings.HasPrefix(part, "min=") {
			// Could parse min values for numbers
		} else if strings.HasPrefix(part, "max=") {
			// Could parse max values for numbers
		}
	}
}
