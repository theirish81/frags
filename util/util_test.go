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
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDurationOrDefault(t *testing.T) {
	t.Run("valid duration", func(t *testing.T) {
		assert.Equal(t, 10*time.Second, ParseDurationOrDefault(StrPtr("10s"), 60*time.Second))
	})
	t.Run("nil duration", func(t *testing.T) {
		assert.Equal(t, 60*time.Second, ParseDurationOrDefault(nil, 60*time.Second))
	})
	t.Run("wrong duration", func(t *testing.T) {
		assert.Equal(t, 60*time.Second, ParseDurationOrDefault(StrPtr("foo"), 60*time.Second))
	})
}

func TestToConcreteValue(t *testing.T) {
	v1 := 6
	v2 := &v1
	v3 := &v2
	assert.Equal(t, 6, ToConcreteValue(reflect.ValueOf(v3)).Interface())
}

func TestFragsContext(t *testing.T) {
	t.Run("cancel error propagation", func(t *testing.T) {
		ctx := NewFragsContext(10 * time.Second)
		ctx.Cancel(errors.New("test error"))
		assert.Equal(t, "context canceled: test error", ctx.Err().Error())
	})
	t.Run("test timeout", func(t *testing.T) {
		ctx := NewFragsContext(10 * time.Millisecond)
		time.Sleep(20 * time.Millisecond)
		assert.Error(t, ctx.Err())
		assert.Equal(t, "context deadline exceeded", ctx.Err().Error())
	})
}
