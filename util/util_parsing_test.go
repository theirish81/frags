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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseJSON(t *testing.T) {
	t.Run("parse map", func(t *testing.T) {
		m, err := ParseJSON(`{"foo": "bar"}`)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"foo": "bar"}, m)
	})
	t.Run("parse array", func(t *testing.T) {
		a, err := ParseJSON(`["foo", "bar"]`)
		assert.NoError(t, err)
		assert.Equal(t, []any{"foo", "bar"}, a)
	})
	t.Run("parse map as bytes", func(t *testing.T) {
		m, err := ParseJSON([]byte(`{"foo": "bar"}`))
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"foo": "bar"}, m)
	})
	t.Run("already a map", func(t *testing.T) {
		m, err := ParseJSON(map[string]any{"foo": "bar"})
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"foo": "bar"}, m)
	})
	t.Run("already an array", func(t *testing.T) {
		a, err := ParseJSON([]any{"foo", "bar"})
		assert.NoError(t, err)
		assert.Equal(t, []any{"foo", "bar"}, a)
	})

}

func TestParseCSV(t *testing.T) {
	t.Run("parse csv", func(t *testing.T) {
		a, err := ParseCSV("foo,bar\nbar,foo")
		assert.NoError(t, err)
		assert.Equal(t, [][]string{{"foo", "bar"}, {"bar", "foo"}}, a)
	})
	t.Run("parse csv as bytes", func(t *testing.T) {
		a, err := ParseCSV([]byte("foo,bar\nbar,foo"))
		assert.NoError(t, err)
		assert.Equal(t, [][]string{{"foo", "bar"}, {"bar", "foo"}}, a)
	})
	t.Run("already a csv", func(t *testing.T) {
		a, err := ParseCSV([][]string{{"foo", "bar"}, {"bar", "foo"}})
		assert.NoError(t, err)
		assert.Equal(t, [][]string{{"foo", "bar"}, {"bar", "foo"}}, a)
	})
}
