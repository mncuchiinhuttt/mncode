package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

func (s *dapSession) ensureConfigured(ctx context.Context) error {
	if s.configured {
		return nil
	}
	if err := s.call(ctx, "configurationDone", map[string]interface{}{}, nil); err != nil {
		return err
	}
	s.configured = true
	return nil
}

func (s *dapSession) execute(ctx context.Context, action string, args map[string]interface{}, workspace string) (string, error) {
	threadID := numberArgument(args, "thread_id", 1)
	switch action {
	case "set_breakpoint":
		file, _ := args["file"].(string)
		line := numberArgument(args, "line", 0)
		if file == "" || line < 1 {
			return "", fmt.Errorf("file and positive line are required for set_breakpoint")
		}
		resolved, err := resolveWorkspacePath(workspace, file, false)
		if err != nil {
			return "", err
		}
		var body json.RawMessage
		if err := s.call(ctx, "setBreakpoints", map[string]interface{}{
			"source":      map[string]string{"path": resolved},
			"breakpoints": []map[string]int{{"line": line}},
		}, &body); err != nil {
			return "", err
		}
		if err := s.ensureConfigured(ctx); err != nil {
			return "", err
		}
		return prettyJSON(body), nil
	case "continue":
		if err := s.ensureConfigured(ctx); err != nil {
			return "", err
		}
		s.discardQueuedEvents()
		var body json.RawMessage
		if err := s.call(ctx, "continue", map[string]int{"threadId": threadID}, &body); err != nil {
			return "", err
		}
		event, err := s.waitEvent(ctx, "stopped", "terminated", "exited")
		if err != nil {
			return "", err
		}
		if event.Event != "stopped" {
			return fmt.Sprintf("debuggee %s", event.Event), nil
		}
		return prettyJSON(body), nil
	case "stack_trace":
		if err := s.ensureConfigured(ctx); err != nil {
			return "", err
		}
		var body json.RawMessage
		if err := s.call(ctx, "stackTrace", map[string]interface{}{"threadId": threadID, "levels": 50}, &body); err != nil {
			return "", err
		}
		return prettyJSON(body), nil
	case "scopes":
		if err := s.ensureConfigured(ctx); err != nil {
			return "", err
		}
		frameID := numberArgument(args, "frame_id", 0)
		if frameID < 1 {
			return "", fmt.Errorf("frame_id is required for scopes")
		}
		var body json.RawMessage
		if err := s.call(ctx, "scopes", map[string]int{"frameId": frameID}, &body); err != nil {
			return "", err
		}
		return prettyJSON(body), nil
	case "variables":
		if err := s.ensureConfigured(ctx); err != nil {
			return "", err
		}
		ref := numberArgument(args, "variables_ref", 0)
		if ref < 1 {
			return "", fmt.Errorf("variables_ref is required for variables")
		}
		var body json.RawMessage
		if err := s.call(ctx, "variables", map[string]int{"variablesReference": ref}, &body); err != nil {
			return "", err
		}
		return prettyJSON(body), nil
	case "evaluate":
		if err := s.ensureConfigured(ctx); err != nil {
			return "", err
		}
		expression, _ := args["expression"].(string)
		if expression == "" {
			return "", fmt.Errorf("expression is required for evaluate")
		}
		frameID := numberArgument(args, "frame_id", 0)
		var body json.RawMessage
		if err := s.call(ctx, "evaluate", map[string]interface{}{"expression": expression, "frameId": frameID, "context": "repl"}, &body); err != nil {
			return "", err
		}
		return prettyJSON(body), nil
	default:
		return "", fmt.Errorf("unsupported debugger action: %s", action)
	}
}
