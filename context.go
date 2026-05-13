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
	"encoding/json"
	"strings"

	"github.com/theirish81/frags/evaluators"
	"github.com/theirish81/frags/scoper"
)

// contextualizePrompt adds the current context to the prompt. This includes the already extracted context, if enabled,
// and optional pre-calls which will be called in this function.
func (r *Runner[T]) contextualizePrompt(prompt string, preCallContext *scoper.KnowledgeNode, session Session, scope evaluators.EvalScope) (string, error) {
	var contextData *scoper.KnowledgeNode
	if session.Context != nil && (session.Context.IsTrue() || session.Context.HasTemplate()) {
		if session.Context.IsTrue() {
			contextDataBytes, err := r.safeMarshalDataStructure(true)
			if err != nil {
				return prompt, err
			}
			contextData = scoper.Node("Context", string(contextDataBytes)).ContentType("application/json")
		}
		if session.Context.HasTemplate() {
			renderedData, err := session.Context.RenderTemplate(scope)
			if err != nil {
				return prompt, err
			}
			contextData = scoper.Node("Context", renderedData)
		}
	}
	outPrompt := ""
	if contextData != nil || preCallContext != nil {
		scopeKnowledge := scoper.Node("Scope", "")
		if contextData != nil {
			scopeKnowledge.AppendChild(contextData)
		}
		if preCallContext != nil {
			scopeKnowledge.AppendChild(preCallContext)
		}
		outPrompt = scopeKnowledge.String()
	}
	outPrompt += "\n" + scoper.Node("Prompt", prompt).Description("The task to execute").String()
	return strings.TrimSpace(outPrompt), nil
}

// preCallCtx returns a string representation of a function call and its result, formatting it in a way that it can be
// correctly read by the LLM. Its main purpose is to be inserted in the prompt as part of the context.
func preCallCtx(call FunctionCaller, res any) *scoper.KnowledgeNode {
	data, _ := json.Marshal(res)
	descr := ""
	if call.Description != nil {
		descr = *call.Description
	}
	return scoper.Node("CallResult", string(data)).Name(call.Name).Description(descr).ContentType("application/json")
}
