package agent

import (
	"encoding/json"
	"mncode/pkg/config"
	"mncode/pkg/provider"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShareGPTJSONPreservesRolesAndToolMetadata(t *testing.T) {
	session := &Session{
		ID: "trajectory-test", WorkspaceDir: t.TempDir(), Config: config.DefaultConfig(),
		History: []provider.Message{
			{Role: provider.RoleSystem, Content: "system"},
			{Role: provider.RoleUser, Content: "inspect"},
			{Role: provider.RoleAssistant, Thinking: "reason", ToolCalls: []provider.ToolCall{{ID: "c1", Name: "view_file"}}},
			{Role: provider.RoleTool, ToolResults: []provider.ToolResult{{ToolCallID: "c1", Name: "view_file", Content: "ok"}}},
		},
	}
	data, err := ShareGPTJSON(session)
	if err != nil {
		t.Fatal(err)
	}
	var exported ShareGPTTrajectory
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatal(err)
	}
	if len(exported.Conversations) != 4 || exported.Conversations[0].From != "system" || exported.Conversations[1].From != "human" {
		t.Fatalf("unexpected conversation roles: %+v", exported.Conversations)
	}
	if !strings.Contains(exported.Conversations[2].Value, "<thinking>") || len(exported.Conversations[2].ToolCalls) != 1 {
		t.Fatalf("assistant metadata was lost: %+v", exported.Conversations[2])
	}
	if len(exported.Conversations[3].ToolResults) != 1 {
		t.Fatalf("tool result metadata was lost: %+v", exported.Conversations[3])
	}
}

func TestExportShareGPTFileUsesPrivateWorkspacePath(t *testing.T) {
	dir := t.TempDir()
	session := &Session{ID: "file-test", WorkspaceDir: dir, History: []provider.Message{{Role: provider.RoleUser, Content: "hello"}}}
	path, err := ExportShareGPTFile(session, "exports/trajectory.json")
	if err != nil {
		t.Fatal(err)
	}
	expectedRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(expectedRoot, "exports", "trajectory.json") {
		t.Fatalf("path = %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %04o, want 0600", info.Mode().Perm())
	}
	if _, err := ExportShareGPTFile(session, "exports/trajectory.json"); err == nil {
		t.Fatal("expected existing destination to be rejected")
	}
}
