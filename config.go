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
	McpServers  McpServers       `json:"mcpServers"`
	Collections ToolsCollections `json:"collections"`
}

func (m ToolsConfig) Tools() Tools {
	tools := Tools{}
	for name, _ := range m.McpServers {
		tools = append(tools, Tool{
			Name: name,
			Type: ToolTypeMCP,
		})
	}
	return tools
}

// CollectionConfig defines the configuration for a collection
type CollectionConfig struct {
	Params   map[string]string `json:"params"`
	Disabled bool              `json:"disabled"`
}

type ToolsCollections map[string]CollectionConfig

// McpServerConfig defines the configuration to connect to a MCP server
type McpServerConfig struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	Cwd       string            `json:"cwd"`
	Transport string            `json:"transport"`
	Url       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Disabled  bool              `json:"disabled"`
}

// McpServers is a map of MCP servers
type McpServers map[string]McpServerConfig
