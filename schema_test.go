package frags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var zero = 0
var one = 1

func TestSchema_GetMaxPhase(t *testing.T) {
	s := Schema{
		Description: "this is a test",
		Properties: map[string]*Schema{
			"p1": {
				Type: "string",
			},
			"p2": {
				Type: "integer",
			},
		},
	}
	if s.GetMaxPhase() != -1 {
		t.Error("expected -1")
	}
	s = Schema{
		Description: "this is a test",
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
	assert.Equal(t, -1, s.GetMaxPhase())
}

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
