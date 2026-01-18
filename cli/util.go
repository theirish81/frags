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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// sliceToMap converts a slice of strings with the key=value format into a map of strings. If ignoreErrors is true,
// strings that do not conform to the format are ignored
func sliceToMap(s []string, ignoreErrors bool) (map[string]any, error) {
	m := make(map[string]any, len(s))
	for _, v := range s {
		if matched, _ := regexp.Match("^[^=]+=[^=]+$", []byte(v)); matched {
			kv := strings.SplitN(v, "=", 2)
			m[kv[0]] = kv[1]
		} else if !ignoreErrors {
			return m, errors.New("invalid parameter format: " + v)
		}

	}
	return m, nil
}

func strPtr(str string) *string {
	return &str
}

func intPtr(i int) *int {
	return &i
}

func printDebugAny(res any) {
	switch reflect.ValueOf(res).Kind() {
	case reflect.Map, reflect.Slice:
		out, _ := json.MarshalIndent(res, "", " ")
		fmt.Println(string(out))
	default:
		fmt.Printf("%v", res)
	}
}
