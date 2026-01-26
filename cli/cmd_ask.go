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
	"time"

	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/schema"
	"github.com/theirish81/frags/util"
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
		toolDefinitions := frags.ToolDefinitions{}
		toolsConfig := frags.ToolsConfig{}
		if toolsEnabled {
			var err error
			toolsConfig, err = readToolsFile()
			toolDefinitions = toolsConfig.AsToolDefinitions()
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
		rx := make([]frags.Resource, 0)
		for _, up := range uploads {
			rx = append(rx, frags.Resource{
				Identifier: up,
			})
		}
		mgr.Sessions = frags.Sessions{
			"default": {
				PrePrompt: util.StrPtrToArray(pp),
				Prompt:    args[0],
				Tools:     toolDefinitions,
				Resources: rx,
			},
		}
		mgr.Schema = &schema.Schema{
			Type:     schema.SchemaObject,
			Required: []string{"answer"},
			Properties: map[string]*schema.Schema{
				"answer": {
					Type:        schema.SchemaString,
					Description: "the answer to the prompt",
					XSession:    strPtr("default"),
					XPhase:      0,
				},
			},
		}
		ctx := util.WithFragsContext(cmd.Context(), 15*time.Minute)
		defer ctx.Cancel()
		out, err := execute(ctx, mgr, make(map[string]any), toolsConfig,
			resources.NewFileResourceLoader("./"), log.NewStreamerLogger(slog.Default(), nil, log.InfoChannelLevel))
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		fmt.Println((*out)["answer"])
	},
}

func init() {
	askCmd.Flags().StringVarP(&prePrompt, "pre-prompt", "", "", "a prompt to run before the AI prompt")
	askCmd.Flags().StringVarP(&systemPrompt, "system-prompt", "", "", "the system prompt")
	askCmd.Flags().StringSliceVarP(&uploads, "upload", "u", []string{}, "file path to upload (can be specified multiple times)")
	askCmd.Flags().BoolVarP(&internetSearch, "internet-search", "i", false, "enable internet search")
	askCmd.Flags().BoolVarP(&toolsEnabled, "tools", "t", false, "enable tools")
}
