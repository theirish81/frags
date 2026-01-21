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
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/avast/retry-go/v5"
)

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

// StrPtr returns a pointer to a string
func StrPtr(s string) *string { return &s }

func Ptr[T any](t T) *T { return &t }

// ToConcreteValue returns the concrete value of a reflect.Value It gives up after 5 levels of indirection as a
// deadlock safety measure.
func ToConcreteValue(rv reflect.Value) reflect.Value {
	i := 0
	for (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && i < 5 {
		i++
		rv = rv.Elem()
	}
	return rv
}

func Retry(ctx context.Context, attempts int, callback func() error) error {
	return retry.New(retry.Attempts(uint(attempts)), retry.Delay(time.Second*5), retry.Context(ctx)).Do(callback)
}

func SetInContext(context any, varName string, value any) error {
	v := reflect.ValueOf(context)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("context deve essere un puntatore non nullo")
	}
	elem := v.Elem()
	valToSet := reflect.ValueOf(value)

	switch elem.Kind() {
	case reflect.Struct:
		field := elem.FieldByName(varName)
		if !field.IsValid() {
			return fmt.Errorf("struct doesn't have a field called '%s'", varName)
		}
		if !field.CanSet() {
			return fmt.Errorf("the field '%s' is not exported and cannot be set", varName)
		}

		if !valToSet.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("value of type %s cannot be set to the field %s", valToSet.Type(), field.Type())
		}
		field.Set(valToSet)

	case reflect.Map:
		if elem.Type().Key().Kind() != reflect.String {
			return errors.New("the context keys are not strings")
		}
		if elem.IsNil() {
			return errors.New("the context is nil")
		}

		elem.SetMapIndex(reflect.ValueOf(varName), valToSet)

	default:
		return fmt.Errorf("unsupported context type: %s", elem.Kind())
	}

	return nil
}
