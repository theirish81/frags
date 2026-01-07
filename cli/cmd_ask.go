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

var internetSearch bool
var toolsEnabled bool

var askCmd = &cobra.Command{
	Use:   "ask <prompt>",
	Short: "Ask a question to the AI, using the current Frags settings and tools.",
	Long: `
Ask a question to the AI, using the current Frags settings and tools. This is a simulation of what plans do,
so it's subject to the limitations imposed by generating structured output.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log := slog.Default()
		toolDefinitions := frags.ToolDefinitions{}
		toolsConfig := frags.ToolsConfig{}
		if toolsEnabled {
			var err error
			toolsConfig, err = readToolsFile()
			if err != nil {
				cmd.PrintErrln(err)
			}
		}
		if internetSearch {
			toolDefinitions = append(toolDefinitions, frags.ToolDefinition{
				Name: "internet_search",
				Type: frags.ToolTypeInternetSearch,
			})
		}
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
				Tools:     toolDefinitions,
				Resources: resources,
			},
		}
		mgr.Schema = &frags.Schema{
			Type:     frags.SchemaObject,
			Required: []string{"answer"},
			Properties: map[string]*frags.Schema{
				"answer": {
					Type:        frags.SchemaString,
					Description: "the answer to the prompt",
					XSession:    strPtr("default"),
					XPhase:      0,
				},
			},
		}
		out, err := execute(cmd.Context(), mgr, make(map[string]any), toolsConfig,
			frags.NewFileResourceLoader("./"), log)
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
	askCmd.Flags().BoolVarP(&internetSearch, "internet-search", "i", false, "Enable internet search")
	askCmd.Flags().BoolVarP(&toolsEnabled, "tools", "t", false, "Enable tools")
}
