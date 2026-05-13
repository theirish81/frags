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

package gemini

import (
	"fmt"

	"google.golang.org/genai"
)

func PartToLoggableText(content *genai.Content) string {
	part := content.Parts[len(content.Parts)-1]
	if part.FunctionCall != nil {
		return fmt.Sprintf("%s(%v)", part.FunctionCall.Name, part.FunctionCall.Args)
	}
	if part.FunctionResponse != nil {
		return fmt.Sprintf("%s --> %v", part.FunctionResponse.Name, part.FunctionResponse.Response)
	}
	return part.Text
}

type FunctionResponseMap map[string]any

func NewFunctionResponseMap(data any, err error) FunctionResponseMap {
	res := FunctionResponseMap{
		"output": data,
	}
	if err != nil {
		res["error"] = err.Error()
	}
	res["system_guidance"] = `
* in case of success, ignore this guidance
* if recoverable failure, retry function calling MAX 3 times per function, adjusting the parameters as necessary.
* if the error is deemed not solvable, not recoverable or we hit the 3rd attempt, respond with the failure reasons, preceded by the flag [FATAL]`
	return res
}
