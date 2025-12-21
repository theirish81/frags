package frags

import (
	"context"
	"encoding/json"
	"time"
)

// Ai is an interface for AI models.
type Ai interface {
	Ask(ctx context.Context, text string, schema *Schema, tools Tools, transformers *Transformers, resources ...ResourceData) ([]byte, error)
	New() Ai
	SetFunctions(functions Functions)
	RunFunction(functionCall FunctionCall, transformers *Transformers) (map[string]any, error)
	SetSystemPrompt(systemPrompt string)
}

// dummyHistoryItem is a history item for testing purposes, to use with DummyAi.
type dummyHistoryItem struct {
	Text      string
	Schema    *Schema
	Resources []ResourceData
}

// DummyAi is a dummy AI model for testing purposes.
type DummyAi struct {
	History []dummyHistoryItem
}

// Ask returns a dummy response for testing purposes.
func (d *DummyAi) Ask(_ context.Context, text string, schema *Schema, _ Tools, _ *Transformers, resources ...ResourceData) ([]byte, error) {
	d.History = append(d.History, dummyHistoryItem{Text: text, Schema: schema, Resources: resources})
	out := map[string]string{}
	for k, _ := range schema.Properties {
		out[k] = text
	}
	time.Sleep(1 * time.Second)
	return json.Marshal(out)
}

func (d *DummyAi) SetFunctions(_ Functions) {}
func (d *DummyAi) SetSystemPrompt(_ string) {}
func (d *DummyAi) RunFunction(_ FunctionCall, _ *Transformers) (map[string]any, error) {
	return nil, nil
}

func (d *DummyAi) New() Ai {
	return &DummyAi{History: make([]dummyHistoryItem, 0)}
}

// NewDummyAi returns a new DummyAi instance.
func NewDummyAi() *DummyAi {
	return &DummyAi{History: make([]dummyHistoryItem, 0)}
}
