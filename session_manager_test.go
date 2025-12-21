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

func TestSessionManager_ResolveSchema(t *testing.T) {
	sm := NewSessionManager()

	ref := "#/components/schemas/Address"
	ref2 := "#/components/schemas/Person"

	sm.Components.Schemas = map[string]Schema{
		"Address": {
			Type: "object",
			Properties: map[string]*Schema{
				"street": {Type: "string"},
				"city":   {Type: "string"},
			},
		},
		"Person": {
			Type: "object",
			Properties: map[string]*Schema{
				"name": {Type: "string"},
				"address": {
					Ref: &ref,
				},
			},
		},
	}

	sm.Schema = Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"person": {
				Ref: &ref2,
			},
		},
	}

	err := sm.ResolveSchema()
	assert.NoError(t, err)

	personSchema := sm.Schema.Properties["person"]
	assert.Equal(t, "object", personSchema.Type)
	assert.NotNil(t, personSchema.Properties["address"])

	addressSchema := personSchema.Properties["address"]
	assert.Equal(t, "object", addressSchema.Type)
	assert.Equal(t, "string", addressSchema.Properties["street"].Type)
}

func TestSessionManager_ResolveSchema_AnyOf(t *testing.T) {
	sm := NewSessionManager()

	ref := "#/components/schemas/Address"

	sm.Components.Schemas = map[string]Schema{
		"Address": {
			Type: "object",
			Properties: map[string]*Schema{
				"street": {Type: "string"},
			},
		},
	}

	sm.Schema = Schema{
		AnyOf: []*Schema{
			{Ref: &ref},
		},
	}

	err := sm.ResolveSchema()
	assert.NoError(t, err)
	assert.NotNil(t, sm.Schema.AnyOf[0])
	assert.Equal(t, "object", sm.Schema.AnyOf[0].Type)
}

func TestSessionManager_ResolveSchema_Items(t *testing.T) {
	sm := NewSessionManager()

	ref := "#/components/schemas/Address"

	sm.Components.Schemas = map[string]Schema{
		"Address": {
			Type: "object",
			Properties: map[string]*Schema{
				"street": {Type: "string"},
			},
		},
	}

	sm.Schema = Schema{
		Type: "array",
		Items: &Schema{
			Ref: &ref,
		},
	}

	err := sm.ResolveSchema()
	assert.NoError(t, err)
	assert.NotNil(t, sm.Schema.Items)
	assert.Equal(t, "object", sm.Schema.Items.Type)
}

func TestSessionManager_ResolveSchema_Circular(t *testing.T) {
	sm := NewSessionManager()

	refA := "#/components/schemas/A"
	refB := "#/components/schemas/B"

	sm.Components.Schemas = map[string]Schema{
		"A": {
			Type: "object",
			Properties: map[string]*Schema{
				"b": {Ref: &refB},
			},
		},
		"B": {
			Type: "object",
			Properties: map[string]*Schema{
				"a": {Ref: &refA},
			},
		},
	}

	sm.Schema = Schema{
		Ref: &refA,
	}

	err := sm.ResolveSchema()
	assert.NoError(t, err)

	aSchema := sm.Schema
	assert.NotNil(t, aSchema.Properties["b"])
	bSchema := aSchema.Properties["b"]
	assert.NotNil(t, bSchema.Properties["a"])
	circularRefSchema := bSchema.Properties["a"]
	assert.NotNil(t, circularRefSchema.Ref)
	assert.Equal(t, refA, *circularRefSchema.Ref)
}

func TestSessionManager_ResolveSchema_NotFound(t *testing.T) {
	sm := NewSessionManager()
	ref := "#/components/schemas/NonExistent"
	sm.Schema = Schema{
		Ref: &ref,
	}
	err := sm.ResolveSchema()
	assert.Error(t, err)
}

func TestSessionManager_ResolveSchema_PreserveXFields(t *testing.T) {
	sm := NewSessionManager()

	ref := "#/components/schemas/Address"
	phase := 1
	session := "test_session"

	sm.Components.Schemas = map[string]Schema{
		"Address": {
			Type: "object",
			Properties: map[string]*Schema{
				"street": {Type: "string"},
			},
		},
	}

	sm.Schema = Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"shipping_address": {
				Ref:      &ref,
				XPhase:   &phase,
				XSession: &session,
			},
		},
	}

	err := sm.ResolveSchema()
	assert.NoError(t, err)

	addressSchema := sm.Schema.Properties["shipping_address"]
	assert.Nil(t, addressSchema.Ref)
	assert.Equal(t, "object", addressSchema.Type)
	assert.Equal(t, phase, *addressSchema.XPhase)
	assert.Equal(t, session, *addressSchema.XSession)
	assert.Equal(t, "string", addressSchema.Properties["street"].Type)
}
