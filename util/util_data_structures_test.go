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

func TestMergeJSON(t *testing.T) {
	s1 := NewProgMap()
	err := s1.MergeJSON([]byte(`{"s2": {"p1":1, "p2": 2}, "arrayOfS2": [{"p1": 3, "p2": 4}]}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(1), s1.GetMap("s2")["p1"])
	assert.Equal(t, float64(3), s1.GetArray("arrayOfS2")[0].(map[string]any)["p1"])

	err = s1.MergeJSON([]byte(`{"s2": {"p1": 99}, "arrayOfS2": [{"p1": 100}]}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(99), s1.GetMap("s2")["p1"])
	assert.Equal(t, float64(3), s1.GetArray("arrayOfS2")[0].(map[string]any)["p1"])
	assert.Equal(t, float64(100), s1.GetArray("arrayOfS2")[1].(map[string]any)["p1"])

	err = s1.MergeJSON([]byte(`{"mapOfS3": { "v1": {"m1": 5, "m2": 6}} }`))
	assert.Nil(t, err)
	assert.Equal(t, float64(99), s1.GetMap("s2")["p1"])
	assert.Equal(t, float64(3), s1.GetArray("arrayOfS2")[0].(map[string]any)["p1"])
	assert.Equal(t, float64(100), s1.GetArray("arrayOfS2")[1].(map[string]any)["p1"])
	assert.Equal(t, float64(5), s1.GetMap("mapOfS3")["v1"].(map[string]any)["m1"])

	err = s1.MergeJSON([]byte(`{"mapOfS3": { "v2": {"m1": 7, "m2": 8}} }`))
	assert.Nil(t, err)
	assert.Equal(t, float64(99), s1.GetMap("s2")["p1"])
	assert.Equal(t, float64(3), s1.GetArray("arrayOfS2")[0].(map[string]any)["p1"])
	assert.Equal(t, float64(100), s1.GetArray("arrayOfS2")[1].(map[string]any)["p1"])
	assert.Equal(t, float64(5), s1.GetMap("mapOfS3")["v1"].(map[string]any)["m1"])
	assert.Equal(t, float64(7), s1.GetMap("mapOfS3")["v2"].(map[string]any)["m1"])

	err = s1.MergeJSON([]byte(`{"mapOfS3": { "v2": {"m1": 70, "m2": 80}} }`))
	assert.Nil(t, err)
	assert.Equal(t, float64(99), s1.GetMap("s2")["p1"])
	assert.Equal(t, float64(3), s1.GetArray("arrayOfS2")[0].(map[string]any)["p1"])
	assert.Equal(t, float64(100), s1.GetArray("arrayOfS2")[1].(map[string]any)["p1"])
	assert.Equal(t, float64(5), s1.GetMap("mapOfS3")["v1"].(map[string]any)["m1"])
	assert.Equal(t, float64(80), s1.GetMap("mapOfS3")["v2"].(map[string]any)["m2"])
}

func TestMergeJSON2(t *testing.T) {
	root := NewProgMap()
	err := root.MergeJSON([]byte(`{"a": {"b": {"c": 1}}}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(1), root.GetMap("a")["b"].(map[string]any)["c"])

	err = root.MergeJSON([]byte(`{"d": {"e": {"f": 2}}}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(1), root.GetMap("a")["b"].(map[string]any)["c"])
	assert.Equal(t, float64(2), root.GetMap("d")["e"].(map[string]any)["f"])

	err = root.MergeJSON([]byte(`{"arr": [1,2,3]}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(1), root.GetMap("a")["b"].(map[string]any)["c"])
	assert.Equal(t, float64(2), root.GetMap("d")["e"].(map[string]any)["f"])
	assert.Equal(t, float64(3), root.GetArray("arr")[2])

	err = root.MergeJSON([]byte(`{"arr": [4,5]}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(1), root.GetMap("a")["b"].(map[string]any)["c"])
	assert.Equal(t, float64(2), root.GetMap("d")["e"].(map[string]any)["f"])
	assert.Equal(t, float64(3), root.GetArray("arr")[2])
	assert.Equal(t, float64(4), root.GetArray("arr")[3])
}

func TestMergeJSON3(t *testing.T) {
	s4 := NewProgMap()
	err := s4.MergeJSON([]byte(`{"Foo": "bar"}`))
	assert.NoError(t, err)
	assert.Equal(t, "bar", s4.GetString("Foo"))
}
