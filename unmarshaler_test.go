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

type s2 struct {
	P1 float64 `json:"p1"`
	P2 float64 `json:"p2"`
}

type s3 struct {
	M1 float64 `json:"m1"`
	M2 float64 `json:"m2"`
}

type s1 struct {
	S2        s2            `json:"s2"`
	ArrayOfS2 []s2          `json:"arrayOfS2"`
	MapOfS3   map[string]s3 `json:"mapOfS3"`
}

func TestMergeJSONInto(t *testing.T) {
	s1 := s1{}
	err := MergeJSONInto(&s1, []byte(`{"s2": {"p1":1, "p2": 2}, "arrayOfS2": [{"p1": 3, "p2": 4}]}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(1), s1.S2.P1)
	assert.Equal(t, float64(3), s1.ArrayOfS2[0].P1)

	err = MergeJSONInto(&s1, []byte(`{"s2": {"p1": 99}, "arrayOfS2": [{"p1": 100}]}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(99), s1.S2.P1)
	assert.Equal(t, float64(3), s1.ArrayOfS2[0].P1)
	assert.Equal(t, float64(100), s1.ArrayOfS2[1].P1)

	err = MergeJSONInto(&s1, []byte(`{"mapOfS3": { "v1": {"m1": 5, "m2": 6}} }`))
	assert.Nil(t, err)
	assert.Equal(t, float64(99), s1.S2.P1)
	assert.Equal(t, float64(3), s1.ArrayOfS2[0].P1)
	assert.Equal(t, float64(100), s1.ArrayOfS2[1].P1)
	assert.Equal(t, float64(5), s1.MapOfS3["v1"].M1)

	err = MergeJSONInto(&s1, []byte(`{"mapOfS3": { "v2": {"m1": 7, "m2": 8}} }`))
	assert.Nil(t, err)
	assert.Equal(t, float64(99), s1.S2.P1)
	assert.Equal(t, float64(3), s1.ArrayOfS2[0].P1)
	assert.Equal(t, float64(100), s1.ArrayOfS2[1].P1)
	assert.Equal(t, float64(5), s1.MapOfS3["v1"].M1)
	assert.Equal(t, float64(7), s1.MapOfS3["v2"].M1)

	err = MergeJSONInto(&s1, []byte(`{"mapOfS3": { "v2": {"m1": 70, "m2": 80}} }`))
	assert.Nil(t, err)
	assert.Equal(t, float64(99), s1.S2.P1)
	assert.Equal(t, float64(3), s1.ArrayOfS2[0].P1)
	assert.Equal(t, float64(100), s1.ArrayOfS2[1].P1)
	assert.Equal(t, float64(5), s1.MapOfS3["v1"].M1)
	assert.Equal(t, float64(80), s1.MapOfS3["v2"].M2)
}

func TestMergeJSONInto2(t *testing.T) {
	root := make(map[string]any)
	err := MergeJSONInto(&root, []byte(`{"a": {"b": {"c": 1}}}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(1), root["a"].(map[string]any)["b"].(map[string]any)["c"])

	err = MergeJSONInto(&root, []byte(`{"d": {"e": {"f": 2}}}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(1), root["a"].(map[string]any)["b"].(map[string]any)["c"])
	assert.Equal(t, float64(2), root["d"].(map[string]any)["e"].(map[string]any)["f"])

	err = MergeJSONInto(&root, []byte(`{"arr": [1,2,3]}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(1), root["a"].(map[string]any)["b"].(map[string]any)["c"])
	assert.Equal(t, float64(2), root["d"].(map[string]any)["e"].(map[string]any)["f"])
	assert.Equal(t, float64(3), root["arr"].([]any)[2])

	err = MergeJSONInto(&root, []byte(`{"arr": [4,5]}`))
	assert.Nil(t, err)
	assert.Equal(t, float64(1), root["a"].(map[string]any)["b"].(map[string]any)["c"])
	assert.Equal(t, float64(2), root["d"].(map[string]any)["e"].(map[string]any)["f"])
	assert.Equal(t, float64(3), root["arr"].([]any)[2])
	assert.Equal(t, float64(4), root["arr"].([]any)[3])

}
