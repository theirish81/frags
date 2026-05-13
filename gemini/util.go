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
