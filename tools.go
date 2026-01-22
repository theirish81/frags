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
	"fmt"
)

type ToolType string

const (
	ToolTypeInternetSearch ToolType = "internet_search"
	ToolTypeFunction       ToolType = "function"
	ToolTypeMCP            ToolType = "mcp"
	ToolTypeCollection     ToolType = "collection"
)

// ToolDefinition defines a tool that can be used in a session. A tool can define a function, an MCP server or a collection.
// Name is either the tool name of the function name
// Collection gets populated during mcp/collection tool breakdown into single functions
// Description is the tool description. Optional, as the tool should already have a description, fill if you wish
// to override the default
// Type is either internet_search, function, mcp or collection
// InputSchema defines the input schema for the tool. mcp and collection tools don't have an input schema.
// Allowlist is a list of allowed functions when the tool is MCP or collection. If nil, all functions are allowed.
type ToolDefinition struct {
	Name        string    `json:"name" yaml:"name"`
	Collection  string    `json:"-" yaml:"-"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Type        ToolType  `json:"type" yaml:"type" validate:"required"`
	InputSchema *Schema   `json:"inputSchema,omitempty" yaml:"inputSchema,omitempty"`
	Allowlist   *[]string `json:"allowlist,omitempty" yaml:"allowlist,omitempty"`
}

func (t ToolDefinition) String() string {
	switch t.Type {
	case ToolTypeInternetSearch:
		return string(ToolTypeInternetSearch)
	case ToolTypeFunction:
		return fmt.Sprintf("%s/%s", t.Type, t.Name)
	case ToolTypeMCP, ToolTypeCollection:
		return fmt.Sprintf("%s/%s", t.Type, t.Name)
	}
	return ""
}

// ToolDefinitions is a list of tools
type ToolDefinitions []ToolDefinition

// HasType returns true if the tool list contains a tool of the given type. This is useful for "special" tools like
// internet_search, in which the type is all it needs.
func (t *ToolDefinitions) HasType(tt ToolType) bool {
	for _, tool := range *t {
		if tool.Type == tt {
			return true
		}
	}
	return false
}

// ToolsCollection is a collection of functions. This is an integration commodity that standardizes how collections
// are defined so that multiple integrations can easily integrate one with the other.
type ToolsCollection interface {
	Name() string
	Description() string
	AsFunctions() Functions
}

type ToolCollections []ToolsCollection
