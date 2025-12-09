package frags

// enrichFirstMessagePrompt adds the current context to the first message prompt.
func (r *Runner[T]) enrichFirstMessagePrompt(prompt string, session Session) (string, error) {
	if session.Context {
		llmContext, err := r.safeMarshalDataStructure(true)
		if err != nil {
			return prompt, err
		}
		prompt = "=== CURRENT CONTEXT ===\n" + string(llmContext) + "\n===\n\n" + prompt
	}
	return prompt, nil
}
