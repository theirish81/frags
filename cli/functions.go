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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/theirish81/frags"
	"github.com/theirish81/fragsfunctions/fs"
	"github.com/theirish81/fragsfunctions/http"
	"github.com/theirish81/fragsfunctions/postgres"
)

// readToolsFile reads the tools configuration file and returns the parsed configuration
func readToolsFile() (frags.ToolsConfig, error) {
	data, err := os.ReadFile("tools.json")
	if errors.Is(err, os.ErrNotExist) {
		data = []byte("{}")
	}
	return parseToolsConfig(data)
}

// parseToolsConfig parses the tools configuration file and returns the parsed configuration
func parseToolsConfig(data []byte) (frags.ToolsConfig, error) {
	config := frags.ToolsConfig{}
	err := json.Unmarshal(data, &config)
	return config, err
}

// connectMcpAndCollections connects to the MCP servers and returns the tools
func connectMcpAndCollections(ctx context.Context, toolsConfig frags.ToolsConfig) (frags.McpTools, []frags.ToolsCollection, frags.ToolDefinitions, frags.Functions, error) {
	mcpTools := make(frags.McpTools, 0)
	toolCollections := make([]frags.ToolsCollection, 0)
	toolDefinitions := make(frags.ToolDefinitions, 0)
	functions := make(frags.Functions, 0)
	toolDefinitions = toolsConfig.AsToolDefinitions()
	mcpTools = toolsConfig.McpServers.McpTools()
	if err := mcpTools.Connect(ctx); err != nil {
		return mcpTools, toolCollections, toolDefinitions, functions, err
	}
	functions, err := mcpTools.AsFunctions(ctx)
	if err != nil {
		return mcpTools, toolCollections, toolDefinitions, functions, err
	}
	for k, v := range toolsConfig.Collections {
		if v.Disabled {
			continue
		}
		switch k {
		case "fs":
			t := fs.New()
			for k, v := range t.AsFunctions() {
				functions[k] = v
			}
			toolCollections = append(toolCollections, t)
		case "postgres":
			c, err := postgres.New(ctx, v.Params["postgres_url"])
			if err != nil {
				return mcpTools, toolCollections, toolDefinitions, functions, err
			}
			for k, v := range c.AsFunctions() {
				functions[k] = v
			}
			toolCollections = append(toolCollections, c)
		case "http":
			c := http.New()
			for k, v := range c.AsFunctions() {
				functions[k] = v
			}
			toolCollections = append(toolCollections, c)
		}
	}
	return mcpTools, toolCollections, toolDefinitions, functions, nil
}
