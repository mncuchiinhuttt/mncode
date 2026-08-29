package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"mncode/pkg/agent"
	"mncode/pkg/provider"
)

type rollbackApprovalUI struct {
	allow bool
	calls int
}

func (u *rollbackApprovalUI) OnQueryStart()                          {}
func (u *rollbackApprovalUI) OnToken(string)                         {}
func (u *rollbackApprovalUI) OnThinking(string)                      {}
func (u *rollbackApprovalUI) OnToolCallStart(*provider.ToolCall)     {}
func (u *rollbackApprovalUI) OnToolCallResult(string, string, bool)  {}
func (u *rollbackApprovalUI) OnSubagentStart(string, string, string) {}
func (u *rollbackApprovalUI) OnSubagentComplete(string, string)      {}
func (u *rollbackApprovalUI) OnGoalDone(string, float64, int, int)   {}
func (u *rollbackApprovalUI) OnError(error)                          {}
func (u *rollbackApprovalUI) Flush()                                 {}
func (u *rollbackApprovalUI) ConfirmToolExecution(call *provider.ToolCall) bool {
	u.calls++
	if call == nil || call.Name != "rollback_checkpoint" {
		return false
	}
	return u.allow
}

func TestHandleUndoCommandPreviewsAndRequiresApproval(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := newUITestRepo(t)
	ui := &rollbackApprovalUI{}
	s := &agent.Session{
		ID:           "ui-undo-session",
		WorkspaceDir: dir,
		History: []provider.Message{
			{Role: provider.RoleUser, Content: "make the edit"},
			{Role: provider.RoleAssistant, Content: "done"},
		},
		UI: ui,
	}
	cp, err := s.CreateTurnCheckpoint(1, "agent edit")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeTurnCheckpoint(cp, "README.md"); err != nil {
		t.Fatal(err)
	}

	HandleUndoCommand([]string{"/undo"}, s)
	if ui.calls != 1 {
		t.Fatalf("approval calls after denial = %d, want 1", ui.calls)
	}
	readUIMustEqual(t, filepath.Join(dir, "README.md"), "agent\n")
	if len(s.History) != 2 {
		t.Fatalf("history changed without approval: %d messages", len(s.History))
	}

	ui.allow = true
	HandleUndoCommand([]string{"/undo"}, s)
	if ui.calls != 2 {
		t.Fatalf("approval calls after approval = %d, want 2", ui.calls)
	}
	readUIMustEqual(t, filepath.Join(dir, "README.md"), "hello\n")
	if len(s.History) != 0 {
		t.Fatalf("history length after approved undo = %d, want 0", len(s.History))
	}
}

func TestHandleCheckpointCommandFinalizesExplicitPaths(t *testing.T) {
	dir := newUITestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "owned.txt"), []byte("owned\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &agent.Session{ID: "ui-checkpoint-session", WorkspaceDir: dir}
	HandleCheckpointCommand([]string{"/checkpoint", "create", "manual", "snapshot", "--paths", "README.md", "owned.txt"}, s)
	list, err := s.ListCheckpoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || !list[0].Completed {
		t.Fatalf("checkpoint list = %#v, want one completed checkpoint", list)
	}
	owned := 0
	for _, entry := range list[0].Manifest {
		if entry.Owned {
			owned++
		}
	}
	if owned != 2 {
		t.Fatalf("owned manifest entries = %d, want 2", owned)
	}
	if !strings.Contains(list[0].Summary, "manual snapshot") {
		t.Fatalf("summary = %q, want explicit summary", list[0].Summary)
	}
}

func newUITestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runUIGit(t, dir, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runUIGit(t, dir, "add", ".")
	runUIGit(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial")
	return dir
}

func runUIGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func readUIMustEqual(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, data, want)
	}
}
