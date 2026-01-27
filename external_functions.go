/*
 * Copyright (C) 2026 Simone Pezzano
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
	"fmt"
	"maps"

	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/schema"
	"github.com/theirish81/frags/util"
)

// ExternalFunction represents a function that can be called by the AI model.
// Name is the function name.
// ToolsCollection is the MCP server or collection that contains the function.
// Description is the function description
// Schema is the input schema for the function.
type ExternalFunction struct {
	Func        func(ctx *util.FragsContext, data map[string]any) (any, error) `yaml:"-"`
	Name        string                                                         `yaml:"name"`
	Collection  string                                                         `yaml:"collection"`
	Description string                                                         `yaml:"description"`
	Schema      *schema.Schema                                                 `yaml:"schema"`
}

func (f ExternalFunction) String() string {
	return fmt.Sprintf("%s:%s", f.Collection, f.Name)
}

// Run runs the function, applying any transformers defined in the runner.
func (f ExternalFunction) Run(ctx *util.FragsContext, args map[string]any, runner ExportableRunner) (any, error) {
	ax, err := runner.Transformers().FilterOnFunctionInput(f.Name).Transform(ctx, maps.Clone(args), runner)
	if err != nil {
		return nil, err
	}
	if !util.IsMapAny(ax) {
		return nil, fmt.Errorf("expected map[string]any, got %T", ax)
	}
	runner.Logger().Debug(log.NewEvent(log.StartEventType, log.FunctionComponent).WithFunction(fmt.Sprintf("%s(%v)", f.Name, ax)))
	ax, err = f.Func(ctx, ax.(map[string]any))
	if err != nil {
		return nil, err
	}

	out, err := runner.Transformers().FilterOnFunctionOutput(f.Name).Transform(ctx, ax, runner)
	if err != nil {
		return nil, err
	}
	runner.Logger().Debug(log.NewEvent(log.EndEventType, log.FunctionComponent).WithFunction(fmt.Sprintf("%s(%v)", f.Name, ax)).WithContent(out))
	return out, nil
}

// ExternalFunctions is a map of functions, indexed by name.
type ExternalFunctions map[string]ExternalFunction

func (f ExternalFunctions) String() string {
	vals := make([]string, 0)
	for _, v := range f {
		vals = append(vals, v.String())
	}
	return fmt.Sprintf("%v", vals)
}

// Get returns a function by name.
func (f ExternalFunctions) Get(name string) ExternalFunction {
	return f[name]
}

// ListByCollection returns a subset of functions, filtered by MCP server or collection
func (f ExternalFunctions) ListByCollection(collection string) ExternalFunctions {
	out := ExternalFunctions{}
	for k, v := range f {
		if v.Collection == collection {
			out[k] = v
		}
	}
	return out
}
