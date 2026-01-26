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
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/theirish81/frags/util"
)

type ValidationError struct {
	Path    string
	Message string
}

type ValidatorOptions struct {
	SoftValidation bool
}

func (e *ValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

// Validate validates the given data against the schema
func (s *Schema) Validate(data any, options *ValidatorOptions) error {
	softValidation := options != nil && options.SoftValidation
	return s.validate(data, "", softValidation)
}

func (s *Schema) validate(data any, path string, softValidation bool) error {
	if data == nil {
		if s.Nullable != nil && *s.Nullable {
			return nil
		}
		return &ValidationError{Path: path, Message: "value is null but schema is not nullable"}
	}

	if len(s.AnyOf) > 0 {
		for _, subSchema := range s.AnyOf {
			if err := subSchema.validate(data, path, softValidation); err == nil {
				return nil
			}
		}
		return &ValidationError{Path: path, Message: "value does not match any of the schemas in anyOf"}
	}

	v := reflect.ValueOf(data)

	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			if s.Nullable != nil && *s.Nullable {
				return nil
			}
			return &ValidationError{Path: path, Message: "value is nil"}
		}
		v = v.Elem()
	}

	switch s.Type {
	case "object":
		return s.validateObject(v, path, softValidation)
	case "array":
		return s.validateArray(v, path, softValidation)
	case "string":
		return s.validateString(v, path)
	case "number", "integer":
		return s.validateNumber(v, path, softValidation)
	case "boolean":
		return s.validateBoolean(v, path, softValidation)
	case "":
		return s.validateByInference(v, path, softValidation)
	default:
		return &ValidationError{Path: path, Message: fmt.Sprintf("unsupported type: %s", s.Type)}
	}
}

func (s *Schema) validateObject(v reflect.Value, path string, softValidation bool) error {
	var m map[string]interface{}

	switch v.Kind() {
	case reflect.Map:
		m = make(map[string]interface{})
		for _, key := range v.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			m[keyStr] = v.MapIndex(key).Interface()
		}
	case reflect.Struct:
		m = structToMap(v)
	default:
		return &ValidationError{Path: path, Message: fmt.Sprintf("expected object, got %s", v.Kind())}
	}

	if s.MinProperties != nil && int64(len(m)) < *s.MinProperties {
		return &ValidationError{Path: path, Message: fmt.Sprintf("object has %d properties, minimum is %d", len(m), *s.MinProperties)}
	}
	if s.MaxProperties != nil && int64(len(m)) > *s.MaxProperties {
		return &ValidationError{Path: path, Message: fmt.Sprintf("object has %d properties, maximum is %d", len(m), *s.MaxProperties)}
	}

	for _, req := range s.Required {
		if _, exists := m[req]; !exists {
			return &ValidationError{Path: path, Message: fmt.Sprintf("missing required property: %s", req)}
		}
	}

	for key, value := range m {
		propPath := path
		if propPath == "" {
			propPath = key
		} else {
			propPath = propPath + "." + key
		}

		if propSchema, exists := s.Properties[key]; exists {
			if err := propSchema.validate(value, propPath, softValidation); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Schema) validateArray(v reflect.Value, path string, softValidation bool) error {
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return &ValidationError{Path: path, Message: fmt.Sprintf("expected array, got %s", v.Kind())}
	}

	length := int64(v.Len())

	if s.MinItems != nil && length < *s.MinItems {
		return &ValidationError{Path: path, Message: fmt.Sprintf("array has %d items, minimum is %d", length, *s.MinItems)}
	}
	if s.MaxItems != nil && length > *s.MaxItems {
		return &ValidationError{Path: path, Message: fmt.Sprintf("array has %d items, maximum is %d", length, *s.MaxItems)}
	}

	if s.Items != nil {
		for i := 0; i < v.Len(); i++ {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if err := s.Items.validate(v.Index(i).Interface(), itemPath, softValidation); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Schema) validateString(v reflect.Value, path string) error {
	if v.Kind() != reflect.String {
		return &ValidationError{Path: path, Message: fmt.Sprintf("expected string, got %s", v.Kind())}
	}

	str := v.String()

	if s.MinLength != nil && int64(len(str)) < *s.MinLength {
		return &ValidationError{Path: path, Message: fmt.Sprintf("string length is %d, minimum is %d", len(str), *s.MinLength)}
	}
	if s.MaxLength != nil && int64(len(str)) > *s.MaxLength {
		return &ValidationError{Path: path, Message: fmt.Sprintf("string length is %d, maximum is %d", len(str), *s.MaxLength)}
	}

	if s.Pattern != "" {
		matched, err := regexp.MatchString(s.Pattern, str)
		if err != nil {
			return &ValidationError{Path: path, Message: fmt.Sprintf("invalid pattern: %v", err)}
		}
		if !matched {
			return &ValidationError{Path: path, Message: fmt.Sprintf("string does not match pattern: %s", s.Pattern)}
		}
	}

	if len(s.Enum) > 0 {
		found := false
		for _, enumVal := range s.Enum {
			if str == enumVal {
				found = true
				break
			}
		}
		if !found {
			return &ValidationError{Path: path, Message: fmt.Sprintf("value must be one of: %v", s.Enum)}
		}
	}

	return nil
}

func (s *Schema) validateNumber(v reflect.Value, path string, softValidation bool) error {
	var num float64
	kind := v.Kind()
	if kind == reflect.String && softValidation {
		if num, err := util.StringValToFloat64(v); err == nil {
			return s.validateNumber(reflect.ValueOf(num), path, softValidation)
		} else {
			return &ValidationError{Path: path, Message: fmt.Sprintf("expected number, got %s", v.Kind())}
		}
	}
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		num = float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		num = float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		num = v.Float()
	default:
		return &ValidationError{Path: path, Message: fmt.Sprintf("expected number, got %s", v.Kind())}
	}

	if s.Type == "integer" && num != float64(int64(num)) {
		return &ValidationError{Path: path, Message: "expected integer, got float"}
	}

	if s.Minimum != nil && num < *s.Minimum {
		return &ValidationError{Path: path, Message: fmt.Sprintf("value %v is less than minimum %v", num, *s.Minimum)}
	}
	if s.Maximum != nil && num > *s.Maximum {
		return &ValidationError{Path: path, Message: fmt.Sprintf("value %v is greater than maximum %v", num, *s.Maximum)}
	}

	return nil
}

func (s *Schema) validateBoolean(v reflect.Value, path string, softValidation bool) error {
	if v.Kind() == reflect.String && softValidation {
		if b, err := util.StringValToToBool(v); err == nil {
			return s.validateBoolean(reflect.ValueOf(b), path, softValidation)
		} else {
			return &ValidationError{Path: path, Message: fmt.Sprintf("expected boolean, got %s", v.Kind())}
		}
	}
	if v.Kind() != reflect.Bool {
		return &ValidationError{Path: path, Message: fmt.Sprintf("expected boolean, got %s", v.Kind())}
	}
	return nil
}

func (s *Schema) validateByInference(v reflect.Value, path string, softValidation bool) error {
	switch v.Kind() {
	case reflect.Map, reflect.Struct:
		return s.validateObject(v, path, softValidation)
	case reflect.Slice, reflect.Array:
		return s.validateArray(v, path, softValidation)
	case reflect.String:
		return s.validateString(v, path)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return s.validateNumber(v, path, softValidation)
	case reflect.Bool:
		return s.validateBoolean(v, path, softValidation)
	default:
		return &ValidationError{Path: path, Message: fmt.Sprintf("unsupported kind: %s", v.Kind())}
	}
}

func structToMap(v reflect.Value) map[string]interface{} {
	m := make(map[string]interface{})
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)

		jsonTag := field.Tag.Get("json")
		name := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
		}

		if !v.Field(i).CanInterface() {
			continue
		}

		m[name] = v.Field(i).Interface()
	}

	return m
}
