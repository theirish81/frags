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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const json1 = `{"p1": "v1", "p2": "v2"}`
const json2 = `{"p3": "v3", "p4": "v4"}`

type s1 struct {
	Foo string
}

func TestProgMap_UnmarshalJSON(t *testing.T) {
	progmap := ProgMap{}
	err := progmap.UnmarshalJSON([]byte(json1))
	assert.Nil(t, err)
	err = progmap.UnmarshalJSON([]byte(json2))
	assert.Nil(t, err)
	assert.Equal(t, ProgMap{"p1": "v1", "p2": "v2", "p3": "v3", "p4": "v4"}, progmap)
}

func TestAnyToResultMap(t *testing.T) {
	t.Run("any input to map", func(t *testing.T) {
		assert.Equal(t, map[string]any{"result": "yay"}, AnyToResultMap("yay"))
	})
	t.Run("map into result map", func(t *testing.T) {
		assert.Equal(t, map[string]any{"foo": "bar"}, AnyToResultMap(map[string]any{"foo": "bar"}))
	})
}

func TestInitDataStructure(t *testing.T) {
	t.Run("init map", func(t *testing.T) {
		v := InitDataStructure[ProgMap]()
		assert.Equal(t, ProgMap{}, *v)
	})

	t.Run("init struct", func(t *testing.T) {

		v := InitDataStructure[s1]()
		assert.Equal(t, s1{}, *v)
	})

}
