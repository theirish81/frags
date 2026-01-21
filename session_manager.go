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
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Session defines an LLM session, with its own context.
// Each session has a Prompt, a NextPhasePrompt for the phases after the first, and a list of resources to load.
// Each session may also have a PrePrompt, that is an LLM interaction that happens before the main one, produces
// no structured data, and has the sole purpose to enrich the context and get it ready. This is mostly useful for
// situations in which we need to use an extraction functionality that poorly harmonizes with a structured output.
// PreCalls defines a list of functions to call before the main interaction.
// PrePrompt is the prompt that will be called before the main interaction. This is mainly for context enrichment
// Prompt defines the main interaction.
// NextPhasePrompt defines the prompt that will be called after the main interaction.
// Resources configure resource loaders to load files for the session.
// Timeout defines the maximum time the session can run for.
// DependsOn defines a list of sessions that must be completed before this session can start, and expressions defining
// code evaluations against the already extracted data, to determine whether the session can start.
// Context defines whether the partially extracted data should be passed to the session.
// Attempts defines the number of times each phase should be retried if it fails.
// ToolDefinitions defines the tools that can be used in this session.
// IterateOn describes a variable (typically a list) over which we will iterate the session. The session will run
// len(IterateOn) times. Use an github.com/expr-lang/expr expression.
// Vars defines variables that are local to the session.
type Session struct {
	PreCalls        FunctionCalls   `json:"preCalls" yaml:"preCalls" validate:"omitempty,dive"`
	PrePrompt       PrePrompt       `json:"prePrompt" yaml:"prePrompt"`
	Prompt          string          `json:"prompt" yaml:"prompt" validate:"omitempty,min=3"`
	NextPhasePrompt string          `json:"nextPhasePrompt" yaml:"nextPhasePrompt"`
	Resources       []Resource      `json:"resources" yaml:"resources" validate:"dive"`
	Timeout         *string         `json:"timeout" yaml:"timeout"`
	DependsOn       Dependencies    `json:"dependsOn" yaml:"dependsOn"`
	Context         bool            `json:"context" yaml:"context"`
	Attempts        int             `json:"attempts" yaml:"attempts"`
	Tools           ToolDefinitions `json:"tools" yaml:"tools"`
	IterateOn       *string         `json:"iterateOn" yaml:"iterateOn"`
	Vars            map[string]any  `json:"vars" yaml:"vars"`
}

type PrePrompt []string

func (p *PrePrompt) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try unmarshaling as a string first
	var single string
	if err := unmarshal(&single); err == nil {
		*p = []string{single}
		return nil
	}

	// Try unmarshaling as an array of strings
	var multi []string
	if err := unmarshal(&multi); err == nil {
		*p = multi
		return nil
	}

	return fmt.Errorf("prePrompt must be a string or array of strings")
}

// RenderPrePrompts renders the pre-prompt (which may contain Go templates), with the given scope
func (s *Session) RenderPrePrompts(scope EvalScope) (PrePrompt, error) {
	if s.PrePrompt == nil || len(s.PrePrompt) == 0 {
		return nil, errors.New("prePrompt is nil or empty")
	}
	renderedPrePrompt := make(PrePrompt, 0)
	for _, p := range s.PrePrompt {
		r, err := EvaluateTemplate(p, scope)
		renderedPrePrompt = append(renderedPrePrompt, r)
		if err != nil {
			return renderedPrePrompt, err
		}
	}

	return renderedPrePrompt, nil
}

func (s *Session) HasPrePrompt() bool {
	return s.PrePrompt != nil && len(s.PrePrompt) > 0
}

func (s *Session) HasPrompt() bool {
	return s.Prompt != ""
}

// RenderPrompt renders the prompt (which may contain Go templates), with the given scope
func (s *Session) RenderPrompt(scope EvalScope) (string, error) {
	return EvaluateTemplate(s.Prompt, scope)
}

// RenderNextPhasePrompt renders the next phase prompt (which may contain Go templat es), with the given scope
func (s *Session) RenderNextPhasePrompt(scope EvalScope) (string, error) {
	return EvaluateTemplate(s.NextPhasePrompt, scope)
}

type ResourceDestination string

const (
	AiResourceDestination        ResourceDestination = "ai"
	VarsResourceDestination      ResourceDestination = "vars"
	PrePromptResourceDestination ResourceDestination = "prePrompt"
	PromptResourceDestination    ResourceDestination = "prompt"
)

// Resource defines a resource to load, with an identifier and a map of parameters
type Resource struct {
	Identifier string               `json:"identifier" yaml:"identifier" validate:"required,min=1"`
	Params     map[string]string    `json:"params" yaml:"params"`
	In         *ResourceDestination `json:"in" yaml:"in" validate:"omitempty,oneof=ai vars prePrompt prompt"`
	Var        *string              `json:"var" yaml:"var"`
}

// Sessions is a map of session IDs to sessions.
type Sessions map[string]Session

// SessionManager manages the LLM sessions and the schema. Sessions split the contribution on the schema
type SessionManager struct {
	Parameters   *ParametersConfig `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Transformers *Transformers     `yaml:"transformers,omitempty" json:"transformers,omitempty"`
	SystemPrompt *string           `yaml:"systemPrompt,omitempty" json:"systemPrompt,omitempty"`
	Components   Components        `yaml:"components" json:"components"`
	Sessions     Sessions          `yaml:"sessions" json:"sessions" validate:"required,min=1,dive"`
	Schema       *Schema           `yaml:"schema,omitempty" json:"schema,omitempty"`
	Vars         map[string]any    `yaml:"vars" json:"vars,omitempty"`
	PreCalls     FunctionCalls     `yaml:"preCalls" json:"preCalls,omitempty"`
}

type Parameter struct {
	Name   string  `yaml:"name" json:"name"`
	Schema *Schema `yaml:"schema" json:"schema"`
}

type Parameters []Parameter

// ParametersConfig holds a list of Parameters and a flag to allow loose type checking. We're using this to allow
// less accurate input mechanisms (like a CLI) to input everything as strings, and still validate it against the
// schema.
type ParametersConfig struct {
	Parameters
	LooseType bool
}

func (p *ParametersConfig) SetLooseType(looseType bool) {
	if p != nil {
		p.LooseType = looseType
	}
}

func (p *ParametersConfig) UnmarshalYAML(node *yaml.Node) error {
	var params Parameters
	if err := node.Decode(&params); err != nil {
		return err
	}
	p.Parameters = params
	return nil
}

// UnmarshalJSON allows unmarshaling a Parameters slice directly into ParametersConfig
func (p *ParametersConfig) UnmarshalJSON(data []byte) error {
	var params Parameters
	if err := json.Unmarshal(data, &params); err != nil {
		return err
	}
	p.Parameters = params
	return nil
}

func (p *ParametersConfig) Validate(data any) error {
	schema := Schema{Type: SchemaObject, Properties: map[string]*Schema{}, Required: make([]string, 0)}
	for _, param := range p.Parameters {
		schema.Required = append(schema.Required, param.Name)
		schema.Properties[param.Name] = param.Schema
	}
	return schema.Validate(data, &ValidatorOptions{SoftValidation: p.LooseType})
}

// Components holds the reusable components of the sessions and schema
type Components struct {
	Prompts map[string]string `yaml:"prompts" json:"prompts"`
	Schemas map[string]Schema `yaml:"schemas" json:"schemas"`
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
	s.Schema = &schema
}

// FromYAML unmarshals a YAML document into the SessionManager.
func (s *SessionManager) FromYAML(data []byte) error {
	return yaml.Unmarshal(data, s)
}

// initNullSchema initializes the schema if it is nil
func (s *SessionManager) initNullSchema() {
	if s.Schema == nil {
		schema := Schema{
			Type:       SchemaObject,
			Properties: map[string]*Schema{},
			Required:   make([]string, 0),
		}
		for k, _ := range s.Sessions {
			schema.Properties[k] = &Schema{
				Type:     SchemaString,
				XSession: StrPtr(k),
				XPhase:   0,
			}
			schema.Required = append(schema.Required, k)
		}
		s.Schema = &schema
	}
}
