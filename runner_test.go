package frags

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type T struct {
	P1 string `json:"p1"`
	P2 string `json:"p2"`
	P3 string `json:"p3"`
	P4 string `json:"p4"`
}

func TestRunner_Run(t *testing.T) {
	sessionData, _ := os.ReadFile("test_data/sessions.yaml")
	mgr := NewSessionManager()
	err := mgr.FromYAML(sessionData)
	assert.Nil(t, err)
	ai := NewDummyAi()
	runner := NewRunner[T](mgr, NewDummyResourceLoader(), ai)
	out, err := runner.Run(map[string]string{"animal": "dog", "animals": "giraffes"})

	assert.Nil(t, err)
	assert.NotEmpty(t, out.P1)
	assert.NotEmpty(t, out.P2)
	assert.NotEmpty(t, out.P3)
	assert.NotEmpty(t, out.P4)

	assert.Equal(t, "extract these images data. Make sure they contain dog", out.P3)
	assert.Equal(t, "also these giraffes", out.P4)

}

func TestRunner_RunDependenciesAndContext(t *testing.T) {
	sessionData, _ := os.ReadFile("test_data/dependant_sessions.yaml")
	mgr := NewSessionManager()
	err := mgr.FromYAML(sessionData)
	assert.Nil(t, err)
	ai := NewDummyAi()
	runner := NewRunner[map[string]string](mgr, NewDummyResourceLoader(), ai, WithSessionWorkers(3))
	out, err := runner.Run(nil)
	assert.Nil(t, err)
	assert.Contains(t, (*out)["summary"], "CURRENT CONTEXT")
	assert.Contains(t, (*out)["summary"], "animal1")
	_, ok := (*out)["nop"]
	assert.False(t, ok)
}
