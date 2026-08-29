package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DAPTool exposes Debug Adapter Protocol operations through Delve. Debugger
// sessions are keyed by session_id and stay alive across tool calls.
type DAPTool struct {
	WorkspaceDir string
	mu           sync.Mutex
	sessions     map[string]*dapSession
}

// Name returns the model-facing debugger tool name.
func (d *DAPTool) Name() string { return "debugger" }

// Description explains supported debugger operations.
func (d *DAPTool) Description() string {
	return "Debug Go programs through Delve's Debug Adapter Protocol: launch, set breakpoints, continue, inspect stack/scopes/variables, and evaluate expressions."
}

// Schema returns the debugger request schema.
func (d *DAPTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type": "string", "enum": []string{"launch", "set_breakpoint", "continue", "stack_trace", "scopes", "variables", "evaluate", "disconnect"},
			},
			"session_id":    map[string]interface{}{"type": "string", "description": "Debugger session identifier; defaults to default."},
			"program":       map[string]interface{}{"type": "string", "description": "Go package or executable path relative to workspace for launch."},
			"file":          map[string]interface{}{"type": "string", "description": "Source path for a breakpoint."},
			"line":          map[string]interface{}{"type": "integer", "minimum": 1},
			"thread_id":     map[string]interface{}{"type": "integer", "minimum": 1},
			"frame_id":      map[string]interface{}{"type": "integer", "minimum": 0},
			"variables_ref": map[string]interface{}{"type": "integer", "minimum": 0},
			"expression":    map[string]interface{}{"type": "string"},
		},
		"required": []string{"action"},
	}
}

// Close disconnects all debugger sessions owned by this tool.
func (d *DAPTool) Close() error {
	d.mu.Lock()
	sessions := make([]*dapSession, 0, len(d.sessions))
	for id, session := range d.sessions {
		sessions = append(sessions, session)
		delete(d.sessions, id)
	}
	d.mu.Unlock()
	var firstErr error
	for _, session := range sessions {
		if err := session.close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Execute performs one DAP operation and returns the adapter response.
func (d *DAPTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	action, _ := args["action"].(string)
	if action == "" {
		return "", fmt.Errorf("action is required")
	}
	sessionID, _ := args["session_id"].(string)
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "default"
	}
	if action == "disconnect" {
		return d.disconnect(sessionID)
	}

	d.mu.Lock()
	if d.sessions == nil {
		d.sessions = make(map[string]*dapSession)
	}
	session := d.sessions[sessionID]
	if session == nil {
		if action != "launch" {
			d.mu.Unlock()
			return "", fmt.Errorf("debug session %q is not launched", sessionID)
		}
		program, err := d.resolveProgram(args)
		if err != nil {
			d.mu.Unlock()
			return "", err
		}
		session, err = startDAP(ctx, d.WorkspaceDir, program)
		if err != nil {
			d.mu.Unlock()
			return "", err
		}
		d.sessions[sessionID] = session
		d.mu.Unlock()
		return "debug session launched", nil
	}
	d.mu.Unlock()

	result, err := session.execute(ctx, action, args, d.WorkspaceDir)
	if err != nil && ctx.Err() != nil {
		d.mu.Lock()
		if d.sessions[sessionID] == session {
			delete(d.sessions, sessionID)
		}
		d.mu.Unlock()
		_ = session.close()
	}
	return result, err
}

func (d *DAPTool) resolveProgram(args map[string]interface{}) (string, error) {
	program, _ := args["program"].(string)
	if strings.TrimSpace(program) == "" {
		return "", fmt.Errorf("program is required for launch")
	}
	resolved, err := resolveWorkspacePath(d.WorkspaceDir, program, false)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(resolved); err != nil || (info.IsDir() && filepath.Ext(resolved) != "") {
		return "", fmt.Errorf("debug program does not exist: %s", program)
	}
	return resolved, nil
}

func (d *DAPTool) disconnect(sessionID string) (string, error) {
	d.mu.Lock()
	session := d.sessions[sessionID]
	delete(d.sessions, sessionID)
	d.mu.Unlock()
	if session == nil {
		return "debug session was not active", nil
	}
	return "debug session disconnected", session.close()
}

func prettyJSON(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return "ok"
	}
	var formatted bytes.Buffer
	if json.Indent(&formatted, raw, "", "  ") == nil {
		return formatted.String()
	}
	return string(raw)
}
