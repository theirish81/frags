package frags

import "gopkg.in/yaml.v3"

// Session defines an LLM session, with its own context.
// Each session has a Prompt, a NextPhasePrompt for the phases after the first, and a list of resources to load.
type Session struct {
	Prompt          string   `json:"prompt" yaml:"prompt"`
	NextPhasePrompt string   `json:"next_phase_prompt" yaml:"nextPhasePrompt"`
	Resources       []string `json:"resources" yaml:"resources"`
	Timeout         *string  `json:"timeout" yaml:"timeout"`
}

// Sessions is a map of session IDs to sessions.
type Sessions map[string]Session

// SessionManager manages the LLM sessions and the schema. Sessions split the contribution on the schema
type SessionManager struct {
	Sessions Sessions `yaml:"sessions" json:"sessions"`
	Schema   Schema   `yaml:"schema" json:"schema"`
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager() SessionManager {
	return SessionManager{Sessions: make(Sessions)}
}

// SetSession sets a session in the SessionManager.
func (s *SessionManager) SetSession(sessionID string, session Session) {
	s.Sessions[sessionID] = session
}

// SetSchema sets the schema in the SessionManager.
func (s *SessionManager) SetSchema(schema Schema) {
	s.Schema = schema
}

// FromYAML unmarshals a YAML document into the SessionManager.
func (s *SessionManager) FromYAML(data []byte) error {
	return yaml.Unmarshal(data, s)
}
