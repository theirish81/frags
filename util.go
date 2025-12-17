package frags

import (
	"encoding/json"
	"reflect"
	"time"
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
