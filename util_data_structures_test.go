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
	"testing"

	"github.com/stretchr/testify/assert"
)

const json1 = `{"p1": "v1", "p2": "v2"}`
const json2 = `{"p3": "v3", "p4": "v4"}`

func TestProgMap_UnmarshalJSON(t *testing.T) {
	progmap := ProgMap{}
	err := progmap.UnmarshalJSON([]byte(json1))
	assert.Nil(t, err)
	err = progmap.UnmarshalJSON([]byte(json2))
	assert.Nil(t, err)
	assert.Equal(t, ProgMap{"p1": "v1", "p2": "v2", "p3": "v3", "p4": "v4"}, progmap)
}
