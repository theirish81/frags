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

package frags

import (
	"context"
	"encoding/json"
	"fmt"
)

// contextualizePrompt adds the current context to the prompt. This includes the already extracted context, if enabled,
// and optional pre-calls which will be called in this function.
func (r *Runner[T]) contextualizePrompt(ctx context.Context, prompt string, session Session) (string, error) {
	// we run the pre-calls first, so that they can be used in the prompt
	preCallsContext, err := r.RunPreCallsToTextContext(ctx, session)
	if err != nil {
		return prompt, err
	}
	prompt = preCallsContext + prompt
	// if the session has context enabled, we add the current context to the prompt
	if session.Context {
		llmContext, err := r.safeMarshalDataStructure(true)
		if err != nil {
			return prompt, err
		}
		prompt = "=== CURRENT CONTEXT ===\n" + string(llmContext) + "\n===\n\n" + prompt
	}
	return prompt, nil
}

// preCallCtx returns a string representation of a function call and its result, formatting it in a way that it can be
// correctly read by the LLM. Its main purpose is to be inserted in the prompt as part of the context.
func preCallCtx(call FunctionCall, res any) string {
	data, _ := json.Marshal(res)
	descr := ""
	if call.Description != nil {
		descr = " - " + *call.Description
	}
	return fmt.Sprintf("\n=== CALL: %s %s ===\n %s \n===\n", call.Name, descr, string(data))
}
