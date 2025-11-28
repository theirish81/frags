package frags

import (
	"gopkg.in/yaml.v3"
)

// Session defines an LLM session, with its own context.
// Each session has a Prompt, a NextPhasePrompt for the phases after the first, and a list of resources to load.
// Resources configure resource loaders to load files for the session.
// Timeout defines the maximum time the session can run for.
// DependsOn defines a list of sessions that must be completed before this session can start, and expressions defining
// code evaluations against the already extracted data, to determine whether the session can start.
// Context defines whether the partially extracted data should be passed to the session
// Attempts defines the number of times each phase should be retried if it fails
type Session struct {
	Prompt          string       `json:"prompt" yaml:"prompt"`
	NextPhasePrompt string       `json:"next_phase_prompt" yaml:"nextPhasePrompt"`
	Resources       []Resource   `json:"resources" yaml:"resources"`
	Timeout         *string      `json:"timeout" yaml:"timeout"`
	DependsOn       Dependencies `json:"depends_on" yaml:"dependsOn"`
	Context         bool         `json:"context" yaml:"context"`
	Attempts        int          `json:"attempts" yaml:"attempts"`
}

// Dependency defines whether this session can run or should:
// * wait on another Session to complete
// * run at all, based on an Expression
type Dependency struct {
	Session    *string `json:"session" yaml:"session"`
	Expression *string `json:"expression" yaml:"expression"`
}

// Dependencies is a list of Dependencies
type Dependencies []Dependency

// RenderPrompt renders the prompt (which may contain Go templat es), with the given scope
func (s *Session) RenderPrompt(scope any) (string, error) {
	return EvaluateTemplate(s.Prompt, scope)
}

// RenderNextPhasePrompt renders the next phase prompt (which may contain Go templat es), with the given scope
func (s *Session) RenderNextPhasePrompt(scope any) (string, error) {
	return EvaluateTemplate(s.NextPhasePrompt, scope)
}

// ListVariables returns a list of all variables used in the prompt and next phase prompt
func (s *Session) ListVariables() []string {
	vars := make([]string, 0)
	vars = append(vars, extractTemplateVariables(s.Prompt)...)
	vars = append(vars, extractTemplateVariables(s.NextPhasePrompt)...)
	return vars
}

// Resource defines a resource to load, with an identifier and a map of parameters
type Resource struct {
	Identifier string            `json:"identifier" yaml:"identifier"`
	Params     map[string]string `json:"params" yaml:"params"`
}

// Sessions is a map of session IDs to sessions.
type Sessions map[string]Session

// ListVariables returns a list of all variables used in the sessions
func (s *Sessions) ListVariables() []string {
	vars := make([]string, 0)
	for _, v := range *s {
		vars = append(vars, v.ListVariables()...)
	}
	return vars
}

// SessionManager manages the LLM sessions and the schema. Sessions split the contribution on the schema
type SessionManager struct {
	Components Components `yaml:"components" json:"components"`
	Sessions   Sessions   `yaml:"sessions" json:"sessions"`
	Schema     Schema     `yaml:"schema" json:"schema"`
}

type Components struct {
	Prompts map[string]string `yaml:"prompts" json:"prompts"`
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
