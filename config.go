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

// ToolsConfig defines the configuration for the MCP clients and collections. This serves no specific purpose
// within Frags itself, but it can be used by integrating applications to standardize the configuration format.
type ToolsConfig struct {
	McpServers  McpServerConfigs       `json:"mcpServers,omitempty"`
	Collections ToolsCollectionConfigs `json:"collections,omitempty"`
}

// AsToolDefinitions returns the tools config as tool definitions
func (t ToolsConfig) AsToolDefinitions() ToolDefinitions {
	return append(t.McpServers.AsToolDefinitions(), t.Collections.AsToolDefinitions()...)
}

// CollectionConfig defines the configuration for a collection
type CollectionConfig struct {
	Params   map[string]string `json:"params,omitempty"`
	Disabled bool              `json:"disabled"`
}

// ToolsCollectionConfigs is a map of collection names to collection configurations
type ToolsCollectionConfigs map[string]CollectionConfig

// AsToolDefinitions returns the collection configs as tool definitions
func (t ToolsCollectionConfigs) AsToolDefinitions() ToolDefinitions {
	tools := ToolDefinitions{}
	for name, _ := range t {
		tools = append(tools, ToolDefinition{
			Name: name,
			Type: ToolTypeCollection,
		})
	}
	return tools
}

// McpServerConfig defines the configuration to connect to a MCP server
type McpServerConfig struct {
	Command      string            `json:"command,omitempty"`
	Args         []string          `json:"args,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	Cwd          string            `json:"cwd,omitempty"`
	Transport    string            `json:"transport,omitempty"`
	Url          string            `json:"url,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Disabled     bool              `json:"disabled"`
	ClientID     *string           `json:"client_id,omitempty"`
	ClientSecret *string           `json:"client_secret,omitempty"`
	Token        *string           `json:"token,omitempty"`
}

// McpServerConfigs is a map of MCP servers
type McpServerConfigs map[string]McpServerConfig

// AsToolDefinitions returns the MCP server configs as tool definitions
func (m McpServerConfigs) AsToolDefinitions() ToolDefinitions {
	tools := ToolDefinitions{}
	for name, _ := range m {
		tools = append(tools, ToolDefinition{
			Name: name,
			Type: ToolTypeMCP,
		})
	}
	return tools
}
