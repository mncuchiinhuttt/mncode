package tools

import (
	"context"
	"fmt"
	"strings"

	"mncode/pkg/hub"
)

// ServiceHubTool allows agents to start, monitor, and manage long-running background processes.
type ServiceHubTool struct {
	DefaultCwd string
}

func (t *ServiceHubTool) Name() string {
	return "service_hub"
}

func (t *ServiceHubTool) Description() string {
	return "Supervised background process manager. Launch dev servers and background services with TCP port and log readiness checks, inspect logs, send stdin, and stop services."
}

func (t *ServiceHubTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"start", "logs", "send", "stop", "ps"},
				"description": "Operation to perform on background services",
			},
			"Name": map[string]interface{}{
				"type":        "string",
				"description": "Unique identifier for the service (e.g. 'web', 'api', 'db')",
			},
			"Command": map[string]interface{}{
				"type":        "string",
				"description": "Executable or command to launch (for 'start' action)",
			},
			"Args": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Arguments passed to the command",
			},
			"ReadyPort": map[string]interface{}{
				"type":        "integer",
				"description": "TCP port that must accept connections before 'start' returns success",
			},
			"ReadyLog": map[string]interface{}{
				"type":        "string",
				"description": "Regex pattern that must appear in stdout before 'start' returns success",
			},
			"TimeoutSec": map[string]interface{}{
				"type":        "integer",
				"description": "Max seconds to wait for readiness (default: 30)",
			},
			"Text": map[string]interface{}{
				"type":        "string",
				"description": "Text to write to service stdin (for 'send' action)",
			},
			"Limit": map[string]interface{}{
				"type":        "integer",
				"description": "Max log lines to retrieve (for 'logs' action, default: 50)",
			},
			"Grep": map[string]interface{}{
				"type":        "string",
				"description": "Regex filter for log lines",
			},
		},
		"required": []string{"Action"},
	}
}

func (t *ServiceHubTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	action, _ := args["Action"].(string)
	mgr := hub.GlobalManager()

	switch strings.ToLower(action) {
	case "start":
		name, _ := args["Name"].(string)
		cmd, _ := args["Command"].(string)
		if name == "" || cmd == "" {
			return "", fmt.Errorf("both 'Name' and 'Command' are required for 'start'")
		}
		var cmdArgs []string
		if rawArgs, ok := args["Args"].([]interface{}); ok {
			for _, a := range rawArgs {
				if s, ok := a.(string); ok {
					cmdArgs = append(cmdArgs, s)
				}
			}
		}
		cwd, _ := args["Cwd"].(string)
		if cwd == "" {
			cwd = t.DefaultCwd
		}
		readyPort := 0
		if rp, ok := args["ReadyPort"].(float64); ok {
			readyPort = int(rp)
		}
		readyLog, _ := args["ReadyLog"].(string)
		timeoutSec := 30
		if to, ok := args["TimeoutSec"].(float64); ok && to > 0 {
			timeoutSec = int(to)
		}

		spec := hub.ServiceSpec{
			Name:        name,
			Command:     cmd,
			Args:        cmdArgs,
			Cwd:         cwd,
			ReadyPort:   readyPort,
			ReadyLog:    readyLog,
			TimeoutSec:  timeoutSec,
		}

		info, err := mgr.Start(ctx, spec)
		if err != nil {
			return "", err
		}
		portStr := ""
		if info.ReadyPort > 0 {
			portStr = fmt.Sprintf(" (Port %d ready)", info.ReadyPort)
		}
		return fmt.Sprintf("Service %q started successfully with PID %d%s.", info.Name, info.PID, portStr), nil

	case "logs":
		name, _ := args["Name"].(string)
		if name == "" {
			return "", fmt.Errorf("'Name' is required for 'logs'")
		}
		limit := 50
		if l, ok := args["Limit"].(float64); ok && l > 0 {
			limit = int(l)
		}
		grep, _ := args["Grep"].(string)
		lines, err := mgr.Logs(name, limit, grep)
		if err != nil {
			return "", err
		}
		if len(lines) == 0 {
			return fmt.Sprintf("No logs found for service %q.", name), nil
		}
		return strings.Join(lines, "\n"), nil

	case "send":
		name, _ := args["Name"].(string)
		text, _ := args["Text"].(string)
		if name == "" {
			return "", fmt.Errorf("'Name' is required for 'send'")
		}
		err := mgr.Send(name, text, true)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Input sent to service %q.", name), nil

	case "stop":
		name, _ := args["Name"].(string)
		if name == "" {
			return "", fmt.Errorf("'Name' is required for 'stop'")
		}
		if err := mgr.Stop(name); err != nil {
			return "", err
		}
		return fmt.Sprintf("Service %q stopped successfully.", name), nil

	case "ps":
		list := mgr.PS()
		if len(list) == 0 {
			return "No active background services.", nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%-14s %-8s %-10s %-8s %s\n", "NAME", "PID", "STATUS", "PORT", "COMMAND"))
		for _, s := range list {
			portStr := "-"
			if s.ReadyPort > 0 {
				portStr = fmt.Sprintf("%d", s.ReadyPort)
			}
			sb.WriteString(fmt.Sprintf("%-14s %-8d %-10s %-8s %s\n", s.Name, s.PID, s.State, portStr, s.Command))
		}
		return sb.String(), nil

	default:
		return "", fmt.Errorf("unknown action %q", action)
	}
}
