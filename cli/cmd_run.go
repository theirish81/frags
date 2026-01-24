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
	"time"

	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
)

var runCmd = &cobra.Command{
	Use:   "run <path/to/plan.yaml>",
	Short: "Run a frags plan from a YAML file.",
	Long:  `Run a frags plan from a YAML file. This is frags CLI core functionality.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// validate flags and input
		if err := validateRunArgs(args); err != nil {
			cmd.PrintErrln(err)
			return
		}

		var streamerLogger *frags.StreamerLogger
		if debug {
			streamerLogger = frags.NewStreamerLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})), nil, frags.DebugChannelLevel)

		} else {
			streamerLogger = frags.NewStreamerLogger(slog.Default(), nil, frags.InfoChannelLevel)
		}

		planData, err := os.ReadFile(args[0])
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		sm := frags.NewSessionManager()
		if err := sm.FromYAML(planData); err != nil {
			cmd.PrintErrln(err)
			return
		}

		toolsConfig, err := readToolsFile()
		if err != nil {
			cmd.PrintErrln(err)
		}
		paramsMap, err := sliceToMap(params, false)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		ctx := frags.WithFragsContext(cmd.Context(), 15*time.Minute)
		defer ctx.Cancel()
		result, err := execute(ctx, sm, paramsMap, toolsConfig,
			frags.NewFileResourceLoader(filepath.Dir(args[0])), streamerLogger)
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
	runCmd.Flags().StringVarP(&format, "format", "f", formatYAML, "output format (yaml, json or template)")
	runCmd.Flags().StringVarP(&output, "output", "o", "", "output file")
	runCmd.Flags().StringVarP(&templatePath, "template", "t", "", "go template file (used with -f template)")
	runCmd.Flags().StringSliceVarP(&params, "param", "p", nil, "a parameter to pass to the plan (can be specified multiple times)")
	runCmd.Flags().BoolVarP(&debug, "debug", "d", false, "enable debug logging")
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
