package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AskTool asks questions to the user with interactive multiple-choice options
type AskTool struct {
	AutoApprove bool
	Prompter    func(question string, options []string, isMultiSelect bool) string
}

func (a *AskTool) Name() string {
	return "ask_question"
}

func (a *AskTool) Description() string {
	return "Ask the user one or more multiple-choice questions to clarify requirements, select design options, or resolve ambiguous choices."
}

func (a *AskTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"questions": map[string]interface{}{
				"type":        "array",
				"description": "List of questions to ask the user.",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"question": map[string]interface{}{
							"type":        "string",
							"description": "The question to ask the user.",
						},
						"options": map[string]interface{}{
							"type":        "array",
							"description": "List of selectable answer options.",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
						"is_multi_select": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether multiple options can be chosen.",
						},
					},
					"required": []string{"question", "options"},
				},
			},
			"Question": map[string]interface{}{
				"type":        "string",
				"description": "Direct question string (fallback format).",
			},
			"Options": map[string]interface{}{
				"type":        "array",
				"description": "Direct options list (fallback format).",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}
}

func (a *AskTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	argBytes, _ := json.Marshal(args)

	// Format 1: questions array
	var structured struct {
		Questions []struct {
			Question      string   `json:"question"`
			Options       []string `json:"options"`
			IsMultiSelect bool     `json:"is_multi_select"`
		} `json:"questions"`
		Question string   `json:"Question"`
		Options  []string `json:"Options"`
	}
	_ = json.Unmarshal(argBytes, &structured)

	if len(structured.Questions) > 0 {
		var responses []string
		for i, q := range structured.Questions {
			ans := a.prompt(q.Question, q.Options, q.IsMultiSelect)
			responses = append(responses, fmt.Sprintf("Q%d: %s -> %s", i+1, q.Question, ans))
		}
		return strings.Join(responses, "\n"), nil
	}

	// Format 2: root Question / Options
	qText := structured.Question
	if qText == "" {
		if val, ok := args["question"].(string); ok {
			qText = val
		}
	}
	if qText == "" {
		return "", fmt.Errorf("Question is required")
	}

	opts := structured.Options
	if len(opts) == 0 {
		if rawOpts, ok := args["options"].([]interface{}); ok {
			for _, o := range rawOpts {
				if s, ok := o.(string); ok {
					opts = append(opts, s)
				}
			}
		}
	}

	answer := a.prompt(qText, opts, false)
	return answer, nil
}

func (a *AskTool) prompt(question string, options []string, isMultiSelect bool) string {
	if a.Prompter != nil {
		return a.Prompter(question, options, isMultiSelect)
	}

	fmt.Printf("\n\033[1;36m[Agent Question]\033[0m %s\n", question)
	if len(options) > 0 {
		for i, opt := range options {
			fmt.Printf("  [%d] %s\n", i+1, opt)
		}
	}
	fmt.Print("> ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if num, err := strconv.Atoi(answer); err == nil && num >= 1 && num <= len(options) {
		return fmt.Sprintf("User selected: %s", options[num-1])
	}
	if answer != "" {
		return fmt.Sprintf("User response: %s", answer)
	}
	return "User skipped this question."
}
