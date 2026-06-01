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

package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ProgMap is a custom map type that allows for incremental unmarshaling of JSON data.
// Instead of replacing the entire map contents during unmarshaling, it merges new key-value
// pairs into the existing map, preserving any entries that aren't overwritten by the incoming JSON.
type ProgMap map[string]any

func NewProgMap() ProgMap {
	return make(ProgMap)
}

func (p ProgMap) GetString(key string) string {
	if s, ok := p[key].(string); ok {
		return s
	}
	return ""
}

func (p ProgMap) GetMap(key string) map[string]any {
	if s, ok := p[key].(map[string]any); ok {
		return s
	}
	return nil
}

func (p ProgMap) GetArray(key string) []any {
	if s, ok := p[key].([]any); ok {
		return s
	}
	return nil
}

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

func (p *ProgMap) MergeJSON(jsonBytes []byte) error {
	mapVal := reflect.ValueOf(*p)
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &rawMap); err != nil {
		return err
	}
	if mapVal.Type().Key().Kind() != reflect.String {
		return errors.New("top-level map must have string keys to merge from JSON")
	}
	for key, raw := range rawMap {
		k := reflect.ValueOf(key)
		existing := mapVal.MapIndex(k)

		if existing.IsValid() {
			ev := existing
			if ev.Kind() == reflect.Interface {
				if ev.IsNil() {
					var incoming interface{}
					if err := json.Unmarshal(raw, &incoming); err != nil {
						return fmt.Errorf("map key %s: %w", key, err)
					}
					mapVal.SetMapIndex(k, reflect.ValueOf(incoming))
					continue
				}

				evElem := ev.Elem()
				switch evElem.Kind() {
				case reflect.Slice, reflect.Array:
					var incomingSlice []interface{}
					if err := json.Unmarshal(raw, &incomingSlice); err == nil {
						newSlice := reflect.AppendSlice(evElem, reflect.ValueOf(incomingSlice))
						mapVal.SetMapIndex(k, reflect.ValueOf(newSlice.Interface()))
						continue
					}
					var single interface{}
					if err := json.Unmarshal(raw, &single); err != nil {
						return fmt.Errorf("map key %s: %w", key, err)
					}
					newSlice := reflect.Append(evElem, reflect.ValueOf(single))
					mapVal.SetMapIndex(k, reflect.ValueOf(newSlice.Interface()))
					continue

				case reflect.Map:
					var incoming map[string]interface{}
					if err := json.Unmarshal(raw, &incoming); err != nil {
						var incomingAny interface{}
						if err2 := json.Unmarshal(raw, &incomingAny); err2 != nil {
							return fmt.Errorf("map key %s: %w", key, err2)
						}
						mapVal.SetMapIndex(k, reflect.ValueOf(incomingAny))
						continue
					}
					existingMap, ok := evElem.Interface().(map[string]interface{})
					if !ok {
						var incomingAny interface{}
						if err := json.Unmarshal(raw, &incomingAny); err != nil {
							return fmt.Errorf("map key %s: %w", key, err)
						}
						mapVal.SetMapIndex(k, reflect.ValueOf(incomingAny))
						continue
					}
					for kk, vv := range incoming {
						existingMap[kk] = vv
					}
					mapVal.SetMapIndex(k, reflect.ValueOf(existingMap))
					continue

				default:
					var incoming interface{}
					if err := json.Unmarshal(raw, &incoming); err != nil {
						return fmt.Errorf("map key %s: %w", key, err)
					}
					mapVal.SetMapIndex(k, reflect.ValueOf(incoming))
					continue
				}
			}

			// FALLBACK
			ev = existing
			if ev.Kind() == reflect.Ptr {
				if ev.IsNil() {
					newElem := reflect.New(ev.Type().Elem())
					if err := unmarshalIntoValue(raw, newElem.Elem()); err != nil {
						return fmt.Errorf("map key %s: %w", key, err)
					}
					mapVal.SetMapIndex(k, newElem)
					continue
				}
				elemVal := ev.Elem()
				switch elemVal.Kind() {
				case reflect.Slice, reflect.Array:
					if err := appendIntoSlice(elemVal, raw); err != nil {
						return fmt.Errorf("map key %s: %w", key, err)
					}
					mapVal.SetMapIndex(k, ev)
					continue
				case reflect.Map:
					if err := mergeIntoMapValue(elemVal, raw); err != nil {
						return fmt.Errorf("map key %s: %w", key, err)
					}
					mapVal.SetMapIndex(k, ev)
					continue
				default:
					if err := unmarshalIntoValue(raw, elemVal); err != nil {
						return fmt.Errorf("map key %s: %w", key, err)
					}
					mapVal.SetMapIndex(k, ev)
					continue
				}
			} else {
				switch ev.Kind() {
				case reflect.Slice, reflect.Array:
					if err := appendIntoSlice(ev, raw); err != nil {
						return fmt.Errorf("map key %s: %w", key, err)
					}
					mapVal.SetMapIndex(k, ev)
					continue
				case reflect.Map:
					if err := mergeIntoMapValue(ev, raw); err != nil {
						return fmt.Errorf("map key %s: %w", key, err)
					}
					mapVal.SetMapIndex(k, ev)
					continue
				default:
					newVal := reflect.New(ev.Type()).Elem()
					if err := unmarshalIntoValue(raw, newVal); err != nil {
						return fmt.Errorf("map key %s: %w", key, err)
					}
					mapVal.SetMapIndex(k, newVal)
					continue
				}
			}
		}

		// KEY NOT PRESENT: create new entry from JSON
		elemType := mapVal.Type().Elem()
		var toSet reflect.Value
		if elemType.Kind() == reflect.Ptr {
			ptr := reflect.New(elemType.Elem())
			if err := unmarshalIntoValue(raw, ptr.Elem()); err != nil {
				return fmt.Errorf("map key %s: %w", key, err)
			}
			toSet = ptr
		} else {
			newElem := reflect.New(elemType).Elem()
			if err := unmarshalIntoValue(raw, newElem); err != nil {
				return fmt.Errorf("map key %s: %w", key, err)
			}
			toSet = newElem
		}
		mapVal.SetMapIndex(k, toSet)
	}
	return nil
}

// appendIntoSlice appends elements contained in raw JSON (which is expected to be a JSON array)
func appendIntoSlice(fv reflect.Value, raw json.RawMessage) error {
	elemType := fv.Type().Elem()

	var rawElems []json.RawMessage
	if err := json.Unmarshal(raw, &rawElems); err == nil {
		for _, r := range rawElems {
			newElem := reflect.New(elemType).Elem()
			if err := unmarshalIntoValue(r, newElem); err != nil {
				return err
			}
			fv.Set(reflect.Append(fv, newElem))
		}
		return nil
	}

	newElem := reflect.New(elemType).Elem()
	if err := unmarshalIntoValue(raw, newElem); err != nil {
		return err
	}
	fv.Set(reflect.Append(fv, newElem))
	return nil
}

// mergeIntoMapValue expects fv to be a map value (reflect.Value of a map).
func mergeIntoMapValue(fv reflect.Value, raw json.RawMessage) error {
	if fv.IsNil() {
		fv.Set(reflect.MakeMap(fv.Type()))
	}
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return fmt.Errorf("expected JSON object to merge into map, got: %w", err)
	}

	if fv.Type().Key().Kind() != reflect.String {
		return errors.New("map keys must be strings to merge from JSON")
	}
	for k, r := range rawMap {
		elemType := fv.Type().Elem()
		var toSet reflect.Value
		if elemType.Kind() == reflect.Ptr {
			ptr := reflect.New(elemType.Elem())
			if err := unmarshalIntoValue(r, ptr.Elem()); err != nil {
				return err
			}
			toSet = ptr
		} else {
			newElem := reflect.New(elemType).Elem()
			if err := unmarshalIntoValue(r, newElem); err != nil {
				return err
			}
			toSet = newElem
		}
		fv.SetMapIndex(reflect.ValueOf(k), toSet)
	}
	return nil
}

// unmarshalIntoValue unmarshals raw into the provided reflect.Value (which must be settable).
func unmarshalIntoValue(raw json.RawMessage, v reflect.Value) error {
	if v.Kind() == reflect.Interface {
		var tmp interface{}
		if err := json.Unmarshal(raw, &tmp); err != nil {
			return err
		}
		v.Set(reflect.ValueOf(tmp))
		return nil
	}

	if !v.CanAddr() {
		return errors.New("value not addressable")
	}
	return json.Unmarshal(raw, v.Addr().Interface())
}

// fieldJSONName returns the JSON name for a struct field using the `json` tag or the field name (lowercased first letter as encoding/json does).
func fieldJSONName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return firstRune(f.Name)
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		return firstRune(f.Name)
	}
	return name
}

func firstRune(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = []rune(string(r[0]))[0]
	return string(r)
}

// EmptyMap is a map[string]any that's initialized with no entries.
var EmptyMap = make(map[string]any)

// IsMapAny returns true if the given data is a map[string]any
func IsMapAny(data any) bool {
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

func StrPtrToArray(strPtr *string) []string {
	if strPtr == nil {
		return nil
	}
	return []string{*strPtr}
}
