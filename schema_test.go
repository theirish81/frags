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

func TestSchema_GetPhase(t *testing.T) {
	s := Schema{
		Description: "this is a test",
		Required:    []string{"p1"},
		Properties: map[string]*Schema{
			"p1": {
				Type:   "string",
				XPhase: 0,
			},
			"p2": {
				Type:   "integer",
				XPhase: 1,
			},
		},
	}
	px1, err := s.GetPhase(0)
	assert.Nil(t, err)
	assert.Equal(t, "string", px1.Properties["p1"].Type)
	assert.Len(t, px1.Properties, 1)
	assert.Equal(t, []string{"p1"}, px1.Required)

	px2, err := s.GetPhase(1)
	assert.Nil(t, err)
	assert.Equal(t, "integer", px2.Properties["p2"].Type)
	assert.Len(t, px2.Properties, 1)
	assert.Equal(t, make([]string, 0), px2.Required)
}

func TestSchema_GetContext(t *testing.T) {
	s := Schema{
		Description: "this is a test",
		Required:    []string{"p1"},
		Properties: map[string]*Schema{
			"p1": {
				Type:     "string",
				XSession: strPtr("foo"),
			},
			"p2": {
				Type:     "integer",
				XSession: strPtr("bar"),
			},
		},
	}

	px1, err := s.GetSession("foo")
	assert.Nil(t, err)
	assert.Equal(t, "string", px1.Properties["p1"].Type)
	assert.Len(t, px1.Properties, 1)
	assert.Equal(t, []string{"p1"}, px1.Required)

	px2, err := s.GetSession("bar")
	assert.Nil(t, err)
	assert.Equal(t, "integer", px2.Properties["p2"].Type)
	assert.Len(t, px2.Properties, 1)
	assert.Equal(t, make([]string, 0), px2.Required)
}

func TestSchema_GetContextGetPhaseCombined(t *testing.T) {
	ctxFoo := "foo"
	ctxBar := "bar"
	s := Schema{
		Description: "this is a test",
		Required:    []string{"p1"},
		Properties: map[string]*Schema{
			"p1": {
				Type:     "string",
				XSession: &ctxFoo,
				XPhase:   0,
			},
			"p2": {
				Type:     "integer",
				XSession: &ctxFoo,
				XPhase:   1,
			},
			"p3": {
				Type:     "string",
				XSession: &ctxBar,
				XPhase:   0,
			},
			"p4": {
				Type:     "integer",
				XSession: &ctxBar,
				XPhase:   1,
			},
		},
	}
	c1, _ := s.GetSession(ctxFoo)
	assert.Len(t, c1.Properties, 2)
	assert.Contains(t, c1.Properties, "p1")
	assert.Contains(t, c1.Properties, "p2")
	assert.NotContains(t, c1.Properties, "p3")
	phase0, _ := c1.GetPhase(0)
	assert.Len(t, phase0.Properties, 1)
	assert.Contains(t, phase0.Properties, "p1")
	assert.NotContains(t, phase0.Properties, "p2")

	c2, _ := s.GetSession(ctxBar)
	assert.Len(t, c1.Properties, 2)
	assert.Contains(t, c2.Properties, "p3")
	assert.Contains(t, c2.Properties, "p4")
	assert.NotContains(t, c2.Properties, "p1")
	phase0, _ = c2.GetPhase(0)
	assert.Len(t, phase0.Properties, 1)
	assert.Contains(t, phase0.Properties, "p3")
	assert.NotContains(t, phase0.Properties, "p4")

	phase1, _ := c2.GetPhase(1)
	assert.Len(t, phase1.Properties, 1)
	assert.Contains(t, phase1.Properties, "p4")
	assert.NotContains(t, phase1.Properties, "p3")
}

func TestSchema_Resolve(t *testing.T) {

	ref := "#/components/schemas/Address"
	ref2 := "#/components/schemas/Person"

	comp := Components{Schemas: map[string]Schema{
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
	},
	}

	schema := Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"person": {
				Ref: &ref2,
			},
		},
	}
	err := schema.Resolve(comp)
	assert.NoError(t, err)

	personSchema := schema.Properties["person"]
	assert.Equal(t, "object", personSchema.Type)
	assert.NotNil(t, personSchema.Properties["address"])

	addressSchema := personSchema.Properties["address"]
	assert.Equal(t, "object", addressSchema.Type)
	assert.Equal(t, "string", addressSchema.Properties["street"].Type)
}

func TestSessionManager_ResolveSchema_AnyOf(t *testing.T) {

	ref := "#/components/schemas/Address"

	comp := Components{Schemas: map[string]Schema{
		"Address": {
			Type: "object",
			Properties: map[string]*Schema{
				"street": {Type: "string"},
			},
		},
	},
	}

	schema := Schema{
		AnyOf: []*Schema{
			{Ref: &ref},
		},
	}

	err := schema.Resolve(comp)
	assert.NoError(t, err)
	assert.NotNil(t, schema.AnyOf[0])
	assert.Equal(t, "object", schema.AnyOf[0].Type)
}

func TestSessionManager_ResolveSchema_Items(t *testing.T) {

	ref := "#/components/schemas/Address"

	comp := Components{Schemas: map[string]Schema{
		"Address": {
			Type: "object",
			Properties: map[string]*Schema{
				"street": {Type: "string"},
			},
		},
	},
	}

	schema := Schema{
		Type: "array",
		Items: &Schema{
			Ref: &ref,
		},
	}

	err := schema.Resolve(comp)
	assert.NoError(t, err)
	assert.NotNil(t, schema.Items)
	assert.Equal(t, "object", schema.Items.Type)
}

func TestSessionManager_ResolveSchema_Circular(t *testing.T) {
	refA := "#/components/schemas/A"
	refB := "#/components/schemas/B"

	comp := Components{Schemas: map[string]Schema{
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
	},
	}

	schema := Schema{
		Ref: &refA,
	}

	err := schema.Resolve(comp)
	assert.NoError(t, err)

	aSchema := schema
	assert.NotNil(t, aSchema.Properties["b"])
	bSchema := aSchema.Properties["b"]
	assert.NotNil(t, bSchema.Properties["a"])
	circularRefSchema := bSchema.Properties["a"]
	assert.NotNil(t, circularRefSchema.Ref)
	assert.Equal(t, refA, *circularRefSchema.Ref)
}

func TestSessionManager_ResolveSchema_NotFound(t *testing.T) {
	ref := "#/components/schemas/NonExistent"
	schema := Schema{
		Ref: &ref,
	}
	err := schema.Resolve(Components{})
	assert.Error(t, err)
}

func TestSessionManager_ResolveSchema_PreserveXFields(t *testing.T) {
	ref := "#/components/schemas/Address"
	phase := 1
	session := "test_session"

	comp := Components{Schemas: map[string]Schema{
		"Address": {
			Type: "object",
			Properties: map[string]*Schema{
				"street": {Type: "string"},
			},
		},
	},
	}

	schema := Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"shipping_address": {
				Ref:      &ref,
				XPhase:   phase,
				XSession: &session,
			},
		},
	}

	err := schema.Resolve(comp)
	assert.NoError(t, err)

	addressSchema := schema.Properties["shipping_address"]
	assert.Nil(t, addressSchema.Ref)
	assert.Equal(t, "object", addressSchema.Type)
	assert.Equal(t, phase, addressSchema.XPhase)
	assert.Equal(t, session, *addressSchema.XSession)
	assert.Equal(t, "string", addressSchema.Properties["street"].Type)
}
