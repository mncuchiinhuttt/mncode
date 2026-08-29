package agent

import (
	"encoding/json"
	"fmt"
	"mncode/pkg/provider"
	"mncode/pkg/tools"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ShareGPTMessage is a ShareGPT-compatible conversation turn with optional
// native tool-call metadata retained for replay and training analysis.
type ShareGPTMessage struct {
	From        string                `json:"from"`
	Value       string                `json:"value"`
	ToolCalls   []provider.ToolCall   `json:"tool_calls,omitempty"`
	ToolResults []provider.ToolResult `json:"tool_results,omitempty"`
}

// ShareGPTTrajectory is a serializable session trajectory and its provenance.
type ShareGPTTrajectory struct {
	SessionID     string            `json:"session_id"`
	Model         string            `json:"model,omitempty"`
	Provider      string            `json:"provider,omitempty"`
	Workspace     string            `json:"workspace,omitempty"`
	ExportedAt    time.Time         `json:"exported_at"`
	Conversations []ShareGPTMessage `json:"conversations"`
}

// ShareGPTJSON converts a session's frozen history into stable, provenance-rich JSON.
func ShareGPTJSON(s *Session) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("session is required")
	}
	history := historySnapshot(s)
	if len(history) == 0 {
		return nil, fmt.Errorf("conversation history is empty")
	}
	sessionID := s.ID
	if sessionID == "" || sessionID == "mncode-main" {
		sessionID = "unsaved-session"
	}
	trajectory := ShareGPTTrajectory{
		SessionID: sessionID, Workspace: s.WorkspaceDir,
		ExportedAt: time.Now().UTC(), Conversations: make([]ShareGPTMessage, 0, len(history)),
	}
	if s.Config != nil {
		trajectory.Model = s.Config.Model
		trajectory.Provider = string(s.Config.Provider)
	}
	for _, message := range history {
		from := shareGPTRole(message.Role)
		value := message.Content
		if message.Thinking != "" {
			value = strings.TrimSpace("<thinking>\n" + message.Thinking + "\n</thinking>\n" + value)
		}
		trajectory.Conversations = append(trajectory.Conversations, ShareGPTMessage{
			From: from, Value: value,
			ToolCalls:   append([]provider.ToolCall(nil), message.ToolCalls...),
			ToolResults: append([]provider.ToolResult(nil), message.ToolResults...),
		})
	}
	return json.MarshalIndent(trajectory, "", "  ")
}

// ExportShareGPTFile writes a private ShareGPT export and returns its path. An
// explicit destination must stay inside the workspace; default exports are
// stored under ~/.mncode/exports with restrictive permissions.
func ExportShareGPTFile(s *Session, destination string) (string, error) {
	data, err := ShareGPTJSON(s)
	if err != nil {
		return "", err
	}
	path, err := trajectoryPath(s, destination)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create trajectory export: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return "", err
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	return path, nil
}

func trajectoryPath(s *Session, destination string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("session is required")
	}
	if strings.TrimSpace(destination) == "" {
		base := GetSessionsDir()
		return filepath.Join(filepath.Dir(base), "exports", fmt.Sprintf("%s-%d.sharegpt.json", safeTrajectoryID(s.ID), time.Now().UTC().UnixNano())), nil
	}
	if strings.TrimSpace(s.WorkspaceDir) == "" {
		return "", fmt.Errorf("an explicit export path requires a workspace")
	}
	return tools.ResolveWorkspacePath(s.WorkspaceDir, destination, true)
}

func safeTrajectoryID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "mncode-main" {
		return "session"
	}
	var builder strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "session"
	}
	return builder.String()
}

func shareGPTRole(role provider.Role) string {
	switch role {
	case provider.RoleSystem:
		return "system"
	case provider.RoleAssistant:
		return "gpt"
	case provider.RoleTool:
		return "tool"
	default:
		return "human"
	}
}
