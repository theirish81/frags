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
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
)

var askCmd = &cobra.Command{
	Use:   "ask <prompt>",
	Short: "Ask a question to the AI, using the current Frags settings and tools.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mcpConfig, err := parseMcpConfig()
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		log := slog.Default()

		ai, err := initAi(log)
		fx, err := prepareMcpFunctions(mcpConfig)
		tools := frags.Tools{}
		for name, _ := range mcpConfig.McpServers {
			tools = append(tools, frags.Tool{
				ServerName: name,
				Type:       frags.ToolTypeMCP,
			})
		}
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		ai.SetFunctions(fx)

		mgr := frags.NewSessionManager()

		var pp *string
		if len(prePrompt) > 0 {
			pp = &prePrompt
		}

		if len(systemPrompt) > 0 {
			mgr.SystemPrompt = &systemPrompt
		}
		resources := make([]frags.Resource, 0)
		for _, up := range uploads {
			resources = append(resources, frags.Resource{
				Identifier: up,
			})
		}
		mgr.Sessions = frags.Sessions{
			"default": {
				PrePrompt: pp,
				Prompt:    args[0],
				Tools:     tools,
				Resources: resources,
			},
		}
		mgr.Schema = &frags.Schema{
			Type:     "object",
			Required: []string{"answer"},
			Properties: map[string]*frags.Schema{
				"answer": {
					Type:        "string",
					Description: "the answer to the prompt",
					XSession:    strPtr("default"),
					XPhase:      0,
				},
			},
		}
		runner := frags.NewRunner[frags.ProgMap](mgr, frags.NewFileResourceLoader("."), ai, frags.WithLogger(log))
		out, err := runner.Run(nil)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		fmt.Println((*out)["answer"])
	},
}

func init() {
	askCmd.Flags().StringVarP(&prePrompt, "pre-prompt", "p", "", "A prompt to run before the AI prompt")
	askCmd.Flags().StringVarP(&systemPrompt, "system-prompt", "s", "", "The system prompt")
	askCmd.Flags().StringSliceVarP(&uploads, "upload", "u", []string{}, "file path to upload (can be specified multiple times)")
}
