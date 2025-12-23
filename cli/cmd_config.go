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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Prints the current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		globalConfig, _ := yaml.Marshal(cfg)
		fmt.Println("==== GLOBAL CONFIG ====")
		fmt.Println(string(globalConfig))

		mcpConfig, err := parseMcpConfig()
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		fx, err := prepareMcpFunctions(mcpConfig)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		tools := frags.Tools{}
		for name, _ := range mcpConfig.McpServers {
			tools = append(tools, frags.Tool{
				ServerName: name,
				Type:       frags.ToolTypeMCP,
			})
		}

		toolsText, _ := yaml.Marshal(tools)
		fmt.Println("==== TOOLS CONFIG ====")
		fmt.Println(string(toolsText))

		functionsText, _ := yaml.Marshal(fx)
		fmt.Println("==== FUNCTIONS CONFIG ====")
		fmt.Println(string(functionsText))
	},
}
