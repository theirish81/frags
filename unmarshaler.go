package frags

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// MergeJSONInto unmarshals jsonBytes into target (which must be a pointer to a struct or pointer to a map).
// Top-level (direct children of the root) slice/array fields are appended to (not replaced).
// Top-level map fields are merged (keys inserted/overwritten).
// Other fields are overwritten.
// Nested objects deeper than the root are unmarshaled normally.
func MergeJSONInto(target interface{}, jsonBytes []byte) error {
	if target == nil {
		return errors.New("target is nil")
	}
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr {
		return errors.New("target must be a pointer")
	}
	if rv.IsNil() {
		return errors.New("target pointer is nil")
	}
	elem := rv.Elem()
	switch elem.Kind() {
	case reflect.Struct:
		return mergeIntoStruct(elem, jsonBytes)
	case reflect.Map:
		return mergeIntoMap(elem, jsonBytes)
	default:
		return fmt.Errorf("unsupported top-level kind: %s (want struct or map)", elem.Kind())
	}
}

// mergeIntoStruct unmarshals top-level JSON object
func mergeIntoStruct(sv reflect.Value, jsonBytes []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &rawMap); err != nil {
		return err
	}

	fieldMap := make(map[string]int)
	st := sv.Type()
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		if f.PkgPath != "" && !f.Anonymous {
			continue
		}
		jsonName := fieldJSONName(f)
		if jsonName == "-" {
			continue
		}
		fieldMap[jsonName] = i
	}

	for key, raw := range rawMap {
		idx, ok := fieldMap[key]
		if !ok {
			continue
		}
		fv := sv.Field(idx)
		if !fv.CanSet() {
			continue
		}

		kind := fv.Kind()
		if kind == reflect.Ptr {
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			fv = fv.Elem()
			kind = fv.Kind()
		}

		switch kind {
		case reflect.Slice, reflect.Array:
			if err := appendIntoSlice(fv, raw); err != nil {
				return fmt.Errorf("field %s: %w", st.Field(idx).Name, err)
			}
		case reflect.Map:
			if err := mergeIntoMapValue(fv, raw); err != nil {
				return fmt.Errorf("field %s: %w", st.Field(idx).Name, err)
			}
		default:
			if err := unmarshalIntoValue(raw, fv); err != nil {
				return fmt.Errorf("field %s: %w", st.Field(idx).Name, err)
			}
		}
	}
	return nil
}

// mergeIntoMap handles target being a map (top-level map[string]T).
func mergeIntoMap(mapVal reflect.Value, jsonBytes []byte) error {
	if mapVal.IsNil() {
		mapVal.Set(reflect.MakeMap(mapVal.Type()))
	}
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
		return lowerFirstRune(f.Name)
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		return lowerFirstRune(f.Name)
	}
	return name
}

func lowerFirstRune(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = []rune(strings.ToLower(string(r[0])))[0]
	return string(r)
}
