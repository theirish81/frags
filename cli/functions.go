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
)

func parseMcpConfig() (frags.McpConfig, error) {
	mcpConfig := frags.McpConfig{}
	if data, err := os.ReadFile("mcp.json"); err == nil {
		if err := json.Unmarshal(data, &mcpConfig); err != nil {
			return mcpConfig, err
		}
	}
	return mcpConfig, nil
}

// prepareMcpFunctions creates a map of MCP functions from the configured servers.
func prepareMcpFunctions(mcpConfig frags.McpConfig) (frags.Functions, error) {
	fx := frags.Functions{}
	for name, mcpServer := range mcpConfig.McpServers {
		tool := frags.NewMcpTool(name)
		if err := tool.Connect(context.Background(), mcpServer); err != nil {
			return fx, err
		}
		functions, err := tool.AsFunctions(context.Background())
		if err != nil {
			return fx, err
		}
		for k, v := range functions {
			fx[k] = v
		}
	}
	return fx, nil
}
