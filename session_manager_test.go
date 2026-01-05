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

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestNewSessionManagerValidate(t *testing.T) {
	t.Run("no sessions", func(t *testing.T) {
		mgr := SessionManager{}
		err := validator.New().Struct(mgr)
		assert.Error(t, err)

		mgr.Sessions = map[string]Session{}
		assert.Error(t, err)
	})
	t.Run("has sessions. No prompt", func(t *testing.T) {
		mgr := SessionManager{
			Sessions: map[string]Session{
				"foo": {},
			},
		}
		err := validator.New().Struct(mgr)
		assert.Error(t, err)
	})
	t.Run("has sessions. Has prompt", func(t *testing.T) {
		mgr := SessionManager{
			Sessions: map[string]Session{
				"foo": {
					Prompt: "foo",
				},
			},
		}
		err := validator.New().Struct(mgr)
		assert.NoError(t, err)
	})
	t.Run("session has broken resources", func(t *testing.T) {
		sx := Session{
			Prompt: "yay",
			Resources: []Resource{
				{
					Identifier: "",
				},
			},
		}
		err := validator.New().Struct(sx)
		assert.Error(t, err)
	})
	t.Run("session has valid resources", func(t *testing.T) {
		sx := Session{
			Prompt: "yay",
			Resources: []Resource{
				{
					Identifier: "foo",
				},
			},
		}
		err := validator.New().Struct(sx)
		assert.NoError(t, err)
	})

}

/*func TestRunner_CheckRequirements(t *testing.T) {
	mgr := NewSessionManager()
	mgr.Sessions["foo"] = Session{Prompt: "foo"}
	mgr.Requirements = []string{"a", "b"}
	r := NewRunner[ProgMap](mgr, NewDummyResourceLoader(), NewDummyAi())
	r.params = nil
	assert.Error(t, r.checkParametersRequirements())

	r.params = map[string]string{"a": "1", "b": "2"}
	assert.NoError(t, r.checkParametersRequirements())

	r.params = struct {
		a string
		b string
	}{a: "1", b: "2"}
	assert.NoError(t, r.checkParametersRequirements())
}
*/
