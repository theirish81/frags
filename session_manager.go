package frags

import "gopkg.in/yaml.v3"

type Session struct {
	Prompt          string   `json:"prompt" yaml:"prompt"`
	NextPhasePrompt string   `json:"next_phase_prompt" yaml:"nextPhasePrompt"`
	Resources       []string `json:"resources" yaml:"resources"`
}

type Sessions map[string]Session

type SessionManager struct {
	Sessions Sessions `yaml:"sessions" json:"sessions"`
	Schema   Schema   `yaml:"schema" json:"schema"`
}

func NewSessionManager() SessionManager {
	return SessionManager{Sessions: make(Sessions)}
}

func (s *SessionManager) SetSession(sessionID string, session Session) {
	s.Sessions[sessionID] = session
}

func (s *SessionManager) SetSchema(schema Schema) {
	s.Schema = schema
}

func (s *SessionManager) FromYAML(data []byte) error {
	return yaml.Unmarshal(data, s)
}
