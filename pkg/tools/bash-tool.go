package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BashTool executes shell commands
type BashTool struct {
	DefaultCwd string
}

func (b *BashTool) Name() string {
	return "run_command"
}

func (b *BashTool) Description() string {
	return "Execute a shell command with a timeout and working directory."
}

func (b *BashTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"CommandLine": map[string]interface{}{
				"type":        "string",
				"description": "The exact shell command to execute.",
			},
			"Cwd": map[string]interface{}{
				"type":        "string",
				"description": "The working directory for command execution.",
			},
			"TimeoutMs": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in milliseconds (default 30000ms).",
			},
		},
		"required": []string{"CommandLine"},
	}
}

func (b *BashTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	cmdStr, _ := args["CommandLine"].(string)
	if cmdStr == "" {
		return "", fmt.Errorf("CommandLine is required")
	}

	cwd, _ := args["Cwd"].(string)
	if cwd == "" {
		cwd = b.DefaultCwd
	}

	timeoutMs := 30000
	if t, ok := args["TimeoutMs"].(float64); ok && t > 0 {
		timeoutMs = int(t)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bash", "-c", cmdStr)
	if cwd != "" {
		cmd.Dir = cwd
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var sb strings.Builder
	if stdout.Len() > 0 {
		sb.WriteString("Output:\n")
		sb.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("Error:\n")
		sb.WriteString(stderr.String())
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return sb.String(), fmt.Errorf("command timed out after %d ms: %w", timeoutMs, err)
		}
		return sb.String(), fmt.Errorf("command failed with exit code: %w\n%s", err, sb.String())
	}

	if sb.Len() == 0 {
		return "Command executed successfully with no output.", nil
	}

	return sb.String(), nil
}
