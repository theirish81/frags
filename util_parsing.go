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
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"strings"
)

// parseJSON parses a JSON string into a map[string]any or a slice of any. If the argument is already a map[string]any
// or a slice of any, it's returned as-is.
func parseJSON(data any) (any, error) {
	switch t := data.(type) {
	case map[string]any:
		return t, nil
	case []any:
		return t, nil
	case []byte:
		mapOut := make(map[string]any)
		if err := json.Unmarshal(t, &mapOut); err == nil {
			return mapOut, nil
		}
		sliceOut := make([]any, 0)
		if err := json.Unmarshal(t, &sliceOut); err == nil {
			return sliceOut, nil
		}
	case string:
		return parseJSON([]byte(t))
	}
	return nil, errors.New("cannot JSON-parse the input")
}

// parseCSV parses a CSV string into a [][]string. If the argument is already a [][]string, it's returned as-is.
// If the argument is a string or a []byte, it's parsed as CSV.
func parseCSV(data any) ([][]string, error) {
	switch t := data.(type) {
	case [][]string:
		return t, nil
	case string:
		reader := csv.NewReader(strings.NewReader(t))
		return reader.ReadAll()
	case []byte:
		reader := csv.NewReader(bytes.NewReader(t))
		return reader.ReadAll()
	}
	return nil, errors.New("cannot parse this from CSV into anything")
}
