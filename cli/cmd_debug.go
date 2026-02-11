/*
 * Copyright (C) 2026 Simone Pezzano
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
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/scriptengines"
	"github.com/theirish81/frags/util"
	"gopkg.in/yaml.v3"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug related commands",
}

func initDebugEnv(ctx context.Context) (*frags.Runner[util.ProgMap], error) {
	ai, err := initAi()
	toolsConfig, err := readToolsFile()
	if err != nil {
		return nil, err
	}
	_, _, _, functions, err := connectMcpAndCollections(ctx, toolsConfig)
	if err != nil {
		return nil, err
	}
	ai.SetFunctions(functions)
	runner := frags.NewRunner[util.ProgMap](frags.NewSessionManager(), resources.NewFileResourceLoader("."), ai,
		frags.WithLogger(log.NewStreamerLogger(slog.Default(), nil, log.DebugChannelLevel)),
		frags.WithScriptEngine(scriptengines.NewJavascriptScriptingEngine()),
		frags.WithExternalFunctions(functions),
	)
	return &runner, nil
}

func loadDebugData() (any, error) {
	var data any = nil
	var err error
	if inputPath != "" {
		content, err := os.ReadFile(inputPath)
		if err != nil {
			return data, err
		}
		if parseInput {
			arr := make([]any, 0)
			err = json.Unmarshal(content, &arr)
			data = arr
			if err != nil {
				object := make(map[string]any)
				err = json.Unmarshal(content, &object)
				data = object
				if err != nil {
					return data, err
				}
			}
		} else {
			data = string(content)
		}
	}
	return data, err
}

var debugScriptCmd = &cobra.Command{
	Use:   "script <path/to/script.js>",
	Short: "Run a JavaScript script in the Frags environment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		code, err := os.ReadFile(args[0])
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		runner, err := initDebugEnv(cmd.Context())
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		data, err := loadDebugData()
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		res, err := runner.ScriptEngine().RunCode(util.WithFragsContext(cmd.Context(), 15*time.Minute), string(code), data, runner)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		if res == nil {
			cmd.PrintErrln("script returned no result")
			return
		}
		printDebugAny(res)

	},
}

var debugTransformerCmd = &cobra.Command{
	Use:   "transformer <path/to/transformer.yaml>",
	Short: "Run a single transformer in a Frags environment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		code, err := os.ReadFile(args[0])
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		transformer := frags.Transformer{}
		if err := yaml.Unmarshal(code, &transformer); err != nil {
			cmd.PrintErrln(err)
			return
		}
		runner, err := initDebugEnv(cmd.Context())
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		data, err := loadDebugData()
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		res, err := transformer.Transform(util.WithFragsContext(cmd.Context(), 15*time.Minute), data, runner)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		printDebugAny(res)
	},
}

func init() {
	debugCmd.PersistentFlags().StringVarP(&inputPath, "input-path", "", "", "input file path, Can be any text, or a JSON map/array")
	debugCmd.PersistentFlags().BoolVarP(&parseInput, "parse-input", "", false, "parse input as JSON")
	debugCmd.AddCommand(debugScriptCmd)
	debugCmd.AddCommand(debugTransformerCmd)
}
