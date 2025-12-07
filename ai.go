package frags

import (
	"context"
	"encoding/json"
	"time"
)

// Ai is an interface for AI models.
type Ai interface {
	Ask(ctx context.Context, text string, schema *Schema, tools Tools, resources ...ResourceData) ([]byte, error)
	New() Ai
	SetFunctions(functions Functions)
}

type Function struct {
	Func        func(data map[string]any) (map[string]any, error)
	Server      string
	Description string
	Schema      *Schema
}
type Functions map[string]Function

func (f Functions) Get(name string) Function {
	return f[name]
}
func (f Functions) ListByServer(server string) Functions {
	out := Functions{}
	for k, v := range f {
		if v.Server == server {
			out[k] = v
		}
	}
	return out
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
func (d *DummyAi) Ask(_ context.Context, text string, schema *Schema, _ Tools, resources ...ResourceData) ([]byte, error) {
	d.History = append(d.History, dummyHistoryItem{Text: text, Schema: schema, Resources: resources})
	out := map[string]string{}
	for k, _ := range schema.Properties {
		out[k] = text
	}
	time.Sleep(1 * time.Second)
	return json.Marshal(out)
}

func (d *DummyAi) SetFunctions(_ Functions) {}

func (d *DummyAi) New() Ai {
	return &DummyAi{History: make([]dummyHistoryItem, 0)}
}

// NewDummyAi returns a new DummyAi instance.
func NewDummyAi() *DummyAi {
	return &DummyAi{History: make([]dummyHistoryItem, 0)}
}
