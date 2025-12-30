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

package chatgpt

import (
	"bytes"
	"encoding/json"
	"errors"
)

type ArgsUnion struct {
	String *string
	Map    map[string]any
}

// --- Unmarshal ---

func (a *ArgsUnion) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)

	// Reset
	a.String = nil
	a.Map = nil

	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}

	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		a.String = &s
		return nil
	}

	// Try map
	var m map[string]any
	if err := json.Unmarshal(data, &m); err == nil {
		a.Map = m
		return nil
	}

	return errors.New("ArgsUnion: value must be string or object")
}

// --- Marshal ---

func (a *ArgsUnion) MarshalJSON() ([]byte, error) {
	switch {
	case a.String != nil:
		return json.Marshal(*a.String)
	case a.Map != nil:
		return json.Marshal(a.Map)
	default:
		return []byte("null"), nil
	}
}
