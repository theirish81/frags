package frags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var zero = 0
var one = 1
var ctxFoo = "foo"
var ctxBar = "bar"

func TestSchema_GetPhase(t *testing.T) {
	s := Schema{
		Description: "this is a test",
		Required:    []string{"p1"},
		Properties: map[string]*Schema{
			"p1": {
				Type:   "string",
				XPhase: &zero,
			},
			"p2": {
				Type:   "integer",
				XPhase: &one,
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
				XSession: &ctxFoo,
			},
			"p2": {
				Type:     "integer",
				XSession: &ctxBar,
			},
		},
	}

	px1, err := s.GetSession(ctxFoo)
	assert.Nil(t, err)
	assert.Equal(t, "string", px1.Properties["p1"].Type)
	assert.Len(t, px1.Properties, 1)
	assert.Equal(t, []string{"p1"}, px1.Required)

	px2, err := s.GetSession(ctxBar)
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
				XPhase:   &zero,
			},
			"p2": {
				Type:     "integer",
				XSession: &ctxFoo,
				XPhase:   &one,
			},
			"p3": {
				Type:     "string",
				XSession: &ctxBar,
				XPhase:   &zero,
			},
			"p4": {
				Type:     "integer",
				XSession: &ctxBar,
				XPhase:   &one,
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
