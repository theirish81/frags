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
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
)

var runCmd = &cobra.Command{
	Use:   "run <path/to/plan.yaml>",
	Short: "Run a frags plan from a YAML file.",
	Long:  "Run a frags plan from a YAML file. This is the main most complex functionality of Frags.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// validate flags and input
		if err := validateRunArgs(args); err != nil {
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

		// read session YAML
		data, err := os.ReadFile(args[0])
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		// build session manager from YAML
		sm := frags.NewSessionManager()
		if err := sm.FromYAML(data); err != nil {
			cmd.PrintErrln(err)
			return
		}
		// parameters can only be strings via CLI, so we tell the parameter validator to enable loose type checking,
		// that is, if a string contains a number, it will be parsed as a number if the schema expects it
		sm.Parameters.SetLooseType(true)

		// global vars can reference environment variables. Here's where we render global vars in case there's a
		// reference to an env var.
		if env, err := sliceToMap(os.Environ(), true); err != nil {
			cmd.PrintErrln(err)
			return
		} else {
			sm.Vars, err = frags.EvaluateMapValues(sm.Vars, frags.NewEvalScope().WithVars(frags.ConvertToMapAny[string](env)))
		}

		ai, err := initAi(log)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		mcpTools, _, _, functions, err := loadMcpAndCollections(cmd.Context())
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		defer func() {
			_ = mcpTools.Close()
		}()
		ai.SetFunctions(functions)
		log.Info("available functions", "functions", functions)
		ch := make(chan frags.ProgressMessage, 10)
		go func() {
			for msg := range ch {
				if msg.Error == nil {
					log.Info(string(msg.Action), "session", msg.Session, "phase", msg.Phase, "iteration", msg.Iteration)
				} else {
					log.Error(string(msg.Action), "session", msg.Session, "phase", msg.Phase, "iteration", msg.Iteration, "error", msg.Error)
				}
			}
		}()

		dir := filepath.Dir(args[0])
		workers := cfg.ParallelWorkers
		if workers <= 0 {
			workers = 1
		}

		runner := frags.NewRunner[frags.ProgMap](
			sm,
			frags.NewFileResourceLoader(dir),
			ai,
			frags.WithSessionWorkers(workers),
			frags.WithLogger(log),
			frags.WithProgressChannel(ch),
			frags.WithUseKFormat(cfg.UseKFormat),
			frags.WithScriptEngine(NewJavascriptScriptingEngine()),
		)

		paramsMap, err := sliceToMap(params, false)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		// execute
		result, err := runner.Run(paramsMap)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		// render output according to the chosen format
		text, err := renderResult(result)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		// write to file or stdout
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
	runCmd.Flags().StringVarP(&format, "format", "f", formatYAML, "Output format (yaml, json or template)")
	runCmd.Flags().StringVarP(&output, "output", "o", "", "Output file")
	runCmd.Flags().StringVarP(&templatePath, "template", "t", "", "Go template file (used with -f template)")
	runCmd.Flags().StringSliceVarP(&params, "param", "p", nil, "InputSchema to pass to the template (used with -f template) in key=value format")
	runCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
}

// validateRunArgs checks basic flag constraints and file existence.
func validateRunArgs(args []string) error {
	if format == formatTemplate && templatePath == "" {
		return fmt.Errorf("template path must be specified when using format=template")
	}
	if _, err := os.Stat(args[0]); err != nil {
		return fmt.Errorf("input file error: %w", err)
	}
	if format != formatYAML && format != formatJSON && format != formatTemplate {
		return fmt.Errorf("unsupported format %q", format)
	}
	return nil
}
