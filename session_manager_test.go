package frags

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSessionManager(t *testing.T) {
	sessionData, _ := os.ReadFile("test_data/sessions.yaml")
	mgr := NewSessionManager()
	err := mgr.FromYAML(sessionData)
	assert.Nil(t, err)
	assert.Len(t, mgr.Schema.Properties, 4)
	assert.Len(t, mgr.Sessions, 2)
	assert.Equal(t, len(mgr.Sessions), len(mgr.Schema.GetSessionsIDs()))
}

func TestSession_RenderPrompt(t *testing.T) {
	session := Session{Prompt: "animal is {{ .animal }}"}
	p, err := session.RenderPrompt(map[string]string{"animal": "dog"})
	assert.Nil(t, err)
	assert.Equal(t, "animal is dog", p)
}
