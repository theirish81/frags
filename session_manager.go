package frags

import "gopkg.in/yaml.v3"

// Session defines an LLM session, with its own context.
// Each session has a Prompt, a NextPhasePrompt for the phases after the first, and a list of resources to load.
// Each session may also have a PrePrompt, that is an LLM interaction that happens before the main one, produces
// no structured data, and has the sole purpose to enrich the context and get it ready. This is mostly useful for
// situations in which we need to use an extraction functionality that poorly harmonizes with a structured output.
// Resources configure resource loaders to load files for the session.
// Timeout defines the maximum time the session can run for.
// DependsOn defines a list of sessions that must be completed before this session can start, and expressions defining
// code evaluations against the already extracted data, to determine whether the session can start.
// Context defines whether the partially extracted data should be passed to the session
// Attempts defines the number of times each phase should be retried if it fails
// Tools defines the tools that can be used in this session
type Session struct {
	PrePrompt       *string      `json:"pre_prompt" yaml:"prePrompt"`
	Prompt          string       `json:"prompt" yaml:"prompt"`
	NextPhasePrompt string       `json:"next_phase_prompt" yaml:"nextPhasePrompt"`
	Resources       []Resource   `json:"resources" yaml:"resources"`
	Timeout         *string      `json:"timeout" yaml:"timeout"`
	DependsOn       Dependencies `json:"depends_on" yaml:"dependsOn"`
	Context         bool         `json:"context" yaml:"context"`
	Attempts        int          `json:"attempts" yaml:"attempts"`
	Tools           Tools        `json:"tools" yaml:"tools"`
	IterateOn       *string      `json:"iterate_on" yaml:"iterateOn"`
}

// RenderPrePrompt renders the pre-prompt (which may contain Go templates), with the given scope
func (s *Session) RenderPrePrompt(scope EvalScope) (*string, error) {
	if s.PrePrompt == nil {
		return nil, nil
	}
	px, err := EvaluateTemplate(*s.PrePrompt, scope)
	return &px, err
}

// RenderPrompt renders the prompt (which may contain Go templates), with the given scope
func (s *Session) RenderPrompt(scope EvalScope) (string, error) {
	return EvaluateTemplate(s.Prompt, scope)
}

// RenderNextPhasePrompt renders the next phase prompt (which may contain Go templat es), with the given scope
func (s *Session) RenderNextPhasePrompt(scope EvalScope) (string, error) {
	return EvaluateTemplate(s.NextPhasePrompt, scope)
}

// Resource defines a resource to load, with an identifier and a map of parameters
type Resource struct {
	Identifier string            `json:"identifier" yaml:"identifier"`
	Params     map[string]string `json:"params" yaml:"params"`
}

// Sessions is a map of session IDs to sessions.
type Sessions map[string]Session

// SessionManager manages the LLM sessions and the schema. Sessions split the contribution on the schema
type SessionManager struct {
	SystemPrompt *string    `yaml:"systemPrompt" json:"system_prompt"`
	Components   Components `yaml:"components" json:"components"`
	Sessions     Sessions   `yaml:"sessions" json:"sessions"`
	Schema       Schema     `yaml:"schema" json:"schema"`
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
