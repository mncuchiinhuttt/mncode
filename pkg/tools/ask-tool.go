package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

// AskTool asks questions to the user in interactive mode
type AskTool struct {
	AutoApprove bool
}

func (a *AskTool) Name() string {
	return "ask_user"
}

func (a *AskTool) Description() string {
	return "Ask the user a clarification question or request specific guidance."
}

func (a *AskTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Question": map[string]interface{}{
				"type":        "string",
				"description": "The question to ask the user.",
			},
		},
		"required": []string{"Question"},
	}
}

func (a *AskTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	question, _ := args["Question"].(string)
	if question == "" {
		return "", fmt.Errorf("Question is required")
	}

	fmt.Printf("\n\033[1;36m[Agent Question]\033[0m %s\n> ", question)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(answer), nil
}
