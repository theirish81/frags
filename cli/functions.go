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
