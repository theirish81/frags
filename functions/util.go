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

package functions

import "errors"

func GetArg[T any](args map[string]any, key string) (*T, error) {
	if v1, ok := args[key]; ok {
		if v2, ok := v1.(T); ok {
			return &v2, nil
		}
		return nil, errors.New(key + " argument is not of the expected type")
	}
	return nil, errors.New(key + " argument is required")
}
