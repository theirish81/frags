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

package fctx

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
