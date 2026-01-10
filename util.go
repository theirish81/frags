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
	"path/filepath"
	"reflect"
	"time"
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

// strPtr returns a pointer to a string
func strPtr(s string) *string { return &s }

// replaceExtension replaces the extension of a filename with a new one
func replaceExtension(filename, newExt string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return filename + newExt
	}
	return filename[:len(filename)-len(ext)] + newExt
}

// toConcreteValue returns the concrete value of a reflect.Value It gives up after 5 levels of indirection as a
// deadlock safety measure.
func toConcreteValue(rv reflect.Value) reflect.Value {
	i := 0
	for (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && i < 5 {
		i++
		rv = rv.Elem()
	}
	return rv
}
