package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckpointRollbackRequiresApprovalAndPreservesUserFiles(t *testing.T) {
	dir := newTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "user.txt"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &Session{ID: "session-checkpoint", WorkspaceDir: dir}
	cp, err := s.CreateTurnCheckpoint(1, "agent run")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent.txt"), []byte("created by agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "user.txt"), []byte("user edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeTurnCheckpoint(cp, "README.md", "agent.txt"); err != nil {
		t.Fatal(err)
	}

	if _, err := s.RollbackCheckpoint(cp.ID, false); !errors.Is(err, ErrRollbackApprovalRequired) {
		t.Fatalf("without approval: got %v, want ErrRollbackApprovalRequired", err)
	}
	readMustEqual(t, filepath.Join(dir, "README.md"), "agent\n")
	readMustEqual(t, filepath.Join(dir, "agent.txt"), "created by agent\n")

	if _, err := s.RollbackCheckpoint(cp.ID, true); err != nil {
		t.Fatal(err)
	}
	readMustEqual(t, filepath.Join(dir, "README.md"), "hello\n")
	if _, err := os.Stat(filepath.Join(dir, "agent.txt")); !os.IsNotExist(err) {
		t.Fatalf("agent-created file still exists: %v", err)
	}
	readMustEqual(t, filepath.Join(dir, "user.txt"), "user edit\n")
}

func TestCheckpointRollbackRefusesPostRunUserChanges(t *testing.T) {
	dir := newTestRepo(t)
	s := &Session{ID: "session-conflict", WorkspaceDir: dir}
	cp, err := s.CreateTurnCheckpoint(2, "agent run")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeTurnCheckpoint(cp, "README.md"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("user changed after run\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := s.PreviewRollbackCheckpoint(cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 1 || plan.Conflicts[0] != "README.md" {
		t.Fatalf("conflicts = %#v, want README.md", plan.Conflicts)
	}
	if _, err := s.RollbackCheckpoint(cp.ID, true); err == nil || !strings.Contains(err.Error(), "user changes detected") {
		t.Fatalf("rollback error = %v, want clear user-change failure", err)
	}
	readMustEqual(t, filepath.Join(dir, "README.md"), "user changed after run\n")
}

func TestCreateTurnCheckpointAssignsDurableSessionIdentity(t *testing.T) {
	dir := newTestRepo(t)
	s := &Session{ID: "mncode-main", WorkspaceDir: dir}
	cp, err := s.CreateTurnCheckpoint(3, "identity")
	if err != nil {
		t.Fatal(err)
	}
	if s.ID == "" || s.ID == "mncode-main" {
		t.Fatalf("session identity = %q, want a durable non-sentinel identity", s.ID)
	}
	if cp.SessionID != s.ID || cp.SessionID == "mncode-main" {
		t.Fatalf("checkpoint session identity = %q, session = %q", cp.SessionID, s.ID)
	}
}

func TestFinalizeTurnCheckpointTracksMultipleAppendedPaths(t *testing.T) {
	dir := newTestRepo(t)
	s := &Session{ID: "session-multiple", WorkspaceDir: dir}
	cp, err := s.CreateTurnCheckpoint(4, "multiple paths")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "first.txt"), []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "second.txt"), []byte("second\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Put the appended path first so a stale pointer to README.md would be
	// exercised when the manifest grows.
	if err := s.FinalizeTurnCheckpoint(cp, "first.txt", "README.md", "second.txt"); err != nil {
		t.Fatal(err)
	}
	owned := make(map[string]CheckpointFile)
	for _, entry := range cp.Manifest {
		if entry.Owned {
			owned[entry.Path] = entry
		}
	}
	for _, path := range []string{"first.txt", "README.md", "second.txt"} {
		entry, ok := owned[path]
		if !ok || !entry.AfterExist || entry.AfterHash == "" {
			t.Fatalf("owned manifest entry for %s = %#v, want finalized after-state", path, entry)
		}
	}
}

func TestRollbackPlanTreatsMissingAfterFileAsConflict(t *testing.T) {
	dir := newTestRepo(t)
	s := &Session{ID: "session-missing", WorkspaceDir: dir}
	cp, err := s.CreateTurnCheckpoint(5, "missing conflict")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeTurnCheckpoint(cp, "README.md"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "README.md")); err != nil {
		t.Fatal(err)
	}
	plan, err := s.PreviewRollbackCheckpoint(cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 1 || plan.Conflicts[0] != "README.md" {
		t.Fatalf("conflicts = %#v, want missing README.md to conflict", plan.Conflicts)
	}
}

func readMustEqual(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, data, want)
	}
}
