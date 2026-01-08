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
	"os"

	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
)

var scriptCmd = &cobra.Command{
	Use:   "script <path/to/script.js>",
	Short: "Run a script (JavaScript) on the scripting engine in the Frags context.",
	Long: `
Run a script (JavaScript) on the scripting engine in the Frags context. The purpose is to allow for a
scripting playground for transformers and scripted tools.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile(args[0])
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		var log *slog.Logger
		if debug {
			log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
		} else {
			log = slog.Default()
		}
		ai, err := initAi(log)
		toolsConfig, err := readToolsFile()
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		mcpTools, _, _, functions, err := connectMcpAndCollections(cmd.Context(), toolsConfig)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		defer func() {
			_ = mcpTools.Close()
		}()
		ai.SetFunctions(functions)
		runner := frags.NewRunner[frags.ProgMap](frags.NewSessionManager(), frags.NewFileResourceLoader("."), ai,
			frags.WithLogger(log),
			frags.WithScriptEngine(NewJavascriptScriptingEngine()),
		)
		res, err := runner.ScriptEngine().RunCode(string(data), nil, &runner)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		text, err := renderResult(res)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		if output != "" {
			if err := os.WriteFile(output, text, 0o644); err != nil {
				cmd.PrintErrln(err)
			}
			return
		}
		fmt.Print(string(text))
	},
}

func init() {
	scriptCmd.Flags().StringVarP(&format, "format", "f", formatYAML, "Output format (yaml, json or template)")
	scriptCmd.Flags().StringVarP(&output, "output", "o", "", "Output file")
}
