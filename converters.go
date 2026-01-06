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
	"fmt"
	"reflect"
	"strconv"
)

func valToFloat64(v reflect.Value) (float64, error) {
	if !v.IsValid() {
		return 0, fmt.Errorf("precondition failed: value is invalid")
	}
	if v.Kind() != reflect.String {
		return 0, fmt.Errorf("precondition failed: expected string, got %s", v.Kind())
	}
	strVal := v.String()

	f, err := strconv.ParseFloat(strVal, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string '%s' to float64: %w", strVal, err)
	}
	return f, nil
}

func valToToBool(v reflect.Value) (bool, error) {
	if !v.IsValid() {
		return false, fmt.Errorf("precondition failed: value is invalid")
	}
	if v.Kind() != reflect.String {
		return false, fmt.Errorf("precondition failed: expected string, got %s", v.Kind())
	}
	b, err := strconv.ParseBool(v.String())
	if err != nil {
		return false, fmt.Errorf("failed to convert string '%s' to bool: %w", v.String(), err)
	}
	return b, nil
}
