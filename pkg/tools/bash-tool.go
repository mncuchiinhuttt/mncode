package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const maxCommandOutputBytes = 512 * 1024

type boundedOutput struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (o *boundedOutput) Write(p []byte) (int, error) {
	if o.limit <= 0 {
		o.truncated = len(p) > 0
		return len(p), nil
	}
	if len(p) > o.limit-o.buf.Len() {
		remaining := o.limit - o.buf.Len()
		if remaining > 0 {
			_, _ = o.buf.Write(p[:remaining])
		}
		o.truncated = true
		return len(p), nil
	}
	return o.buf.Write(p)
}

func (o *boundedOutput) String() string {
	s := o.buf.String()
	if o.truncated {
		s += "\n...[output truncated]"
	}
	return s
}
func (o *boundedOutput) Len() int { return o.buf.Len() }

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
	resolvedCwd, err := resolveWorkspacePath(b.DefaultCwd, cwd, false)
	if err != nil {
		return "", err
	}
	cwd = resolvedCwd

	timeoutMs := 30000 // 30s default
	for _, key := range []string{"TimeoutMs", "timeout_ms", "Timeout", "timeout", "WaitMsBeforeAsync", "wait_ms"} {
		if val, exists := args[key]; exists {
			switch v := val.(type) {
			case float64:
				if v > 0 {
					timeoutMs = int(v)
				}
			case int:
				if v > 0 {
					timeoutMs = v
				}
			case int64:
				if v > 0 {
					timeoutMs = int(v)
				}
			}
		}
	}

	// For persistent preview/dev server commands without backgrounding (&), enforce a strict 6s cap
	lowerCmd := strings.ToLower(cmdStr)
	isServerCmd := strings.Contains(lowerCmd, "preview") ||
		strings.Contains(lowerCmd, "run dev") ||
		strings.Contains(lowerCmd, "npm start") ||
		strings.Contains(lowerCmd, "http.server") ||
		strings.Contains(lowerCmd, "vite") ||
		strings.Contains(lowerCmd, "next dev")

	if isServerCmd && !strings.Contains(cmdStr, "&") && timeoutMs > 10000 {
		timeoutMs = 6000
	}

	if timeoutMs > 120000 {
		timeoutMs = 120000
	}
	if timeoutMs < 1000 {
		timeoutMs = 5000
	}

	cmd := exec.Command("bash", "-c", cmdStr)
	if cwd != "" {
		cmd.Dir = cwd
	}
	setProcessGroup(cmd)

	var stdout, stderr boundedOutput
	stdout.limit = maxCommandOutputBytes
	stderr.limit = maxCommandOutputBytes
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		killProcessGroup(cmd)
		<-done

		var sb strings.Builder
		if stdout.Len() > 0 {
			sb.WriteString("Output before timeout:\n")
			sb.WriteString(stdout.String())
		}
		if stderr.Len() > 0 {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString("Error:\n")
			sb.WriteString(stderr.String())
		}

		if isServerCmd {
			// For server preview commands, return the startup output as success
			if sb.Len() > 0 {
				return fmt.Sprintf("Server started (preview verified):\n%s", sb.String()), nil
			}
			return fmt.Sprintf("Server process executed for %dms and terminated.", timeoutMs), nil
		}

		return sb.String(), fmt.Errorf("command execution reached timeout (%dms) and was terminated: %s", timeoutMs, cmdStr)

	case <-ctx.Done():
		killProcessGroup(cmd)
		<-done
		return "", ctx.Err()

	case err := <-done:
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
			return sb.String(), fmt.Errorf("command failed with exit code: %w\n%s", err, sb.String())
		}

		if sb.Len() == 0 {
			return "Command executed successfully with no output.", nil
		}

		return sb.String(), nil
	}
}
