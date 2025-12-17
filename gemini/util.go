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
