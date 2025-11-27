package frags

import (
	"bytes"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Session defines an LLM session, with its own context.
// Each session has a Prompt, a NextPhasePrompt for the phases after the first, and a list of resources to load.
type Session struct {
	Prompt          string       `json:"prompt" yaml:"prompt"`
	NextPhasePrompt string       `json:"next_phase_prompt" yaml:"nextPhasePrompt"`
	Resources       []Resource   `json:"resources" yaml:"resources"`
	Timeout         *string      `json:"timeout" yaml:"timeout"`
	DependsOn       Dependencies `json:"depends_on" yaml:"dependsOn"`
}

type Dependency struct {
	Session    *string `json:"session" yaml:"session"`
	Expression *string `json:"expression" yaml:"expression"`
}

type Dependencies []Dependency

func (s *Session) RenderPrompt(scope any) (string, error) {
	if scope == nil || !strings.Contains(s.Prompt, "{{") {
		return s.Prompt, nil
	}
	tmpl := template.New("tpl")
	parsedTmpl, err := tmpl.Parse(s.Prompt)
	if err != nil {
		return s.Prompt, err
	}
	writer := bytes.NewBufferString("")
	err = parsedTmpl.Execute(writer, scope)
	return writer.String(), err
}

func (s *Session) RenderNextPhasePrompt(scope any) (string, error) {
	if !strings.Contains(s.NextPhasePrompt, "{{") {
		return s.Prompt, nil
	}
	tmpl := template.New("tpl")
	parsedTmpl, err := tmpl.Parse(s.NextPhasePrompt)
	if err != nil {
		return s.Prompt, err
	}
	writer := bytes.NewBufferString("")
	err = parsedTmpl.Execute(writer, scope)
	return writer.String(), err
}

func (s *Session) ListVariables() []string {
	vars := make([]string, 0)
	vars = append(vars, extractTemplateVariables(s.Prompt)...)
	vars = append(vars, extractTemplateVariables(s.NextPhasePrompt)...)
	return vars
}

type Resource struct {
	Identifier string            `json:"identifier" yaml:"identifier"`
	Params     map[string]string `json:"params" yaml:"params"`
}

// Sessions is a map of session IDs to sessions.
type Sessions map[string]Session

func (s *Sessions) ListVariables() []string {
	vars := make([]string, 0)
	for _, v := range *s {
		vars = append(vars, v.ListVariables()...)
	}
	return vars
}

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
