package frags

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Ai interface {
	Ask(ctx context.Context, text string, schema Schema, resources ...Resource) ([]byte, error)
}

type dummyHistoryItem struct {
	Text      string
	Schema    Schema
	Resources []Resource
}

type DummyAi struct {
	History []dummyHistoryItem
}

func (d *DummyAi) Ask(_ context.Context, text string, schema Schema, resources ...Resource) ([]byte, error) {
	d.History = append(d.History, dummyHistoryItem{Text: text, Schema: schema, Resources: resources})
	out := map[string]string{}
	for k, _ := range schema.Properties {
		out[k] = fmt.Sprintf("%s-%v", k, time.Now())
	}
	time.Sleep(1 * time.Second)
	return json.Marshal(out)
}

func NewDummyAi() *DummyAi {
	return &DummyAi{History: make([]dummyHistoryItem, 0)}
}
