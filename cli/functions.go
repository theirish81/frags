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
	"os"

	"github.com/theirish81/frags"
	"github.com/theirish81/fragsfunctions/fs"
	"github.com/theirish81/fragsfunctions/http"
	"github.com/theirish81/fragsfunctions/postgres"
)

func parseToolsConfig() (frags.ToolsConfig, error) {
	toolsConfig := frags.ToolsConfig{}
	if data, err := os.ReadFile("tools.json"); err == nil {
		if err := json.Unmarshal(data, &toolsConfig); err != nil {
			return toolsConfig, err
		}
	}
	return toolsConfig, nil
}

func loadMcpAndCollections(ctx context.Context) (frags.McpTools, []frags.ToolsCollection, frags.ToolDefinitions, frags.Functions, error) {
	mcpConfig, err := parseToolsConfig()
	mcpTools := make(frags.McpTools, 0)
	toolCollections := make([]frags.ToolsCollection, 0)
	toolDefinitions := make(frags.ToolDefinitions, 0)
	functions := make(frags.Functions, 0)
	if err != nil {
		return mcpTools, toolCollections, toolDefinitions, functions, err
	}
	toolDefinitions = mcpConfig.AsToolDefinitions()
	mcpTools = mcpConfig.McpServers.McpTools()
	if err = mcpTools.Connect(ctx); err != nil {
		return mcpTools, toolCollections, toolDefinitions, functions, err
	}
	functions, err = mcpTools.AsFunctions(ctx)
	if err != nil {
		return mcpTools, toolCollections, toolDefinitions, functions, err
	}
	for k, v := range mcpConfig.Collections {
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
