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
func preCallCtx(call FunctionCall, res map[string]any) string {
	data, _ := json.Marshal(res)
	descr := ""
	if call.Description != nil {
		descr = " - " + *call.Description
	}
	return fmt.Sprintf("\n=== CALL: %s %s ===\n %s \n===\n", call.Name, descr, string(data))
}
