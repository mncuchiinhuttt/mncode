package tools

import (
	"context"
	"fmt"
)

// SubagentInvoker represents a callback function to run a subagent
type SubagentInvoker func(ctx context.Context, agentName, prompt string) (string, error)

// SubagentTool allows the agent to delegate tasks to specialized ClaudeKit subagents
type SubagentTool struct {
	Invoker SubagentInvoker
}

func (s *SubagentTool) Name() string {
	return "invoke_subagent"
}

func (s *SubagentTool) Description() string {
	return "Invoke a specialized subagent (e.g. planner, researcher, code-reviewer, tester, debugger, docs-manager) to handle a dedicated task."
}

func (s *SubagentTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"AgentName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the subagent to invoke (e.g. 'planner', 'researcher', 'code-reviewer', 'tester', 'debugger').",
			},
			"Prompt": map[string]interface{}{
				"type":        "string",
				"description": "Actionable instructions and context for the subagent.",
			},
		},
		"required": []string{"AgentName", "Prompt"},
	}
}

func (s *SubagentTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if s.Invoker == nil {
		return "", fmt.Errorf("subagent runner is not configured")
	}

	agentName, _ := args["AgentName"].(string)
	prompt, _ := args["Prompt"].(string)
	if agentName == "" || prompt == "" {
		return "", fmt.Errorf("AgentName and Prompt are required")
	}

	return s.Invoker(ctx, agentName, prompt)
}
