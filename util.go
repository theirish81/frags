package frags

import (
	"encoding/json"
	"regexp"
	"strings"
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

// extractTemplateVariables extracts all template variables from a given string.
func extractTemplateVariables(templateStr string) []string {
	re := regexp.MustCompile(`{{\s*\.([\w.]+)\s*}}`)
	matches := re.FindAllStringSubmatch(templateStr, -1)
	var variables []string
	for _, match := range matches {
		if len(match) > 1 {
			cleanVar := strings.TrimSpace(match[1])
			variables = append(variables, cleanVar)
		}
	}
	return variables
}
