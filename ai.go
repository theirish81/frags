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
	"time"

	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/schema"
	"github.com/theirish81/frags/util"
)

// Ai is an interface for AI models.
type Ai interface {
	Ask(ctx *util.FragsContext, text string, schema *schema.Schema, tools ToolDefinitions, runner ExportableRunner, resources ...resources.ResourceData) ([]byte, error)
	New() Ai
	SetFunctions(functions ExternalFunctions)
	RunFunction(ctx *util.FragsContext, functionCall FunctionCaller, runner ExportableRunner) (any, error)
	SetSystemPrompt(systemPrompt string)
}

// dummyHistoryItem is a history item for testing purposes, to use with DummyAi.
type dummyHistoryItem struct {
	Text      string
	Schema    *schema.Schema
	Resources resources.ResourceDataItems
}

// DummyAi is a dummy AI model for testing purposes.
type DummyAi struct {
	History []dummyHistoryItem
}

// Ask returns a dummy response for testing purposes.
func (d *DummyAi) Ask(_ *util.FragsContext, text string, schema *schema.Schema, _ ToolDefinitions, _ ExportableRunner, resources ...resources.ResourceData) ([]byte, error) {
	d.History = append(d.History, dummyHistoryItem{Text: text, Schema: schema, Resources: resources})
	out := map[string]string{}
	for k, _ := range schema.Properties {
		out[k] = text
	}
	time.Sleep(1 * time.Second)
	return json.Marshal(out)
}

func (d *DummyAi) SetFunctions(_ ExternalFunctions) {}
func (d *DummyAi) SetSystemPrompt(_ string)         {}
func (d *DummyAi) RunFunction(_ *util.FragsContext, _ FunctionCaller, _ ExportableRunner) (any, error) {
	return nil, nil
}

func (d *DummyAi) New() Ai {
	return &DummyAi{History: make([]dummyHistoryItem, 0)}
}

// NewDummyAi returns a new DummyAi instance.
func NewDummyAi() *DummyAi {
	return &DummyAi{History: make([]dummyHistoryItem, 0)}
}
