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
	"log/slog"
	"os"
	"strings"

	"github.com/samber/lo"
	"github.com/theirish81/frags"
)

// execute executes the plan using the specified parameters
func execute(ctx context.Context, sm frags.SessionManager, paramsMap map[string]any, toolConfig frags.ToolsConfig,
	rl frags.ResourceLoader, log *slog.Logger) (*frags.ProgMap, error) {
	// parameters can only be strings via CLI, so we tell the parameter validator to enable loose type checking,
	// that is, if a string contains a number, it will be parsed as a number if the schema expects it
	sm.Parameters.SetLooseType(true)

	// global vars can reference environment variables, as long as the environment variable has the FRAGS_ prefix.
	//Here's where we render global vars in case there's a reference to an env var.
	if env, err := sliceToMap(lo.Filter(os.Environ(), func(item string, index int) bool {
		return strings.HasPrefix(item, "FRAGS_")
	}), true); err != nil {
		return nil, err
	} else {
		sm.Vars, err = frags.EvaluateMapValues(sm.Vars, frags.NewEvalScope().WithVars(env))
	}

	ai, err := initAi(log)
	if err != nil {
		return nil, err
	}
	mcpTools, _, _, functions, err := connectMcpAndCollections(ctx, toolConfig)
	if err != nil {
		return nil, err
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

	workers := cfg.ParallelWorkers
	if workers <= 0 {
		workers = 1
	}

	runner := frags.NewRunner[frags.ProgMap](
		sm,
		rl,
		ai,
		frags.WithSessionWorkers(workers),
		frags.WithLogger(log),
		frags.WithProgressChannel(ch),
		frags.WithUseKFormat(cfg.UseKFormat),
		frags.WithScriptEngine(NewJavascriptScriptingEngine()),
	)

	// execute
	return runner.Run(paramsMap)

}
