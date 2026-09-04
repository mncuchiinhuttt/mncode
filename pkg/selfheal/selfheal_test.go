package selfheal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type mockHealListener struct {
	steps []string
}

func (l *mockHealListener) OnHealStep(step, status string) {
	l.steps = append(l.steps, step)
}

func TestAutoHealCoordinatorLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-selfheal-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	targetFile := filepath.Join(tempDir, "state.txt")
	// State 1: buggy (contains "bug")
	_ = os.WriteFile(targetFile, []byte("buggy"), 0644)

	// Command fails if file contains "buggy"
	reproCmd := "grep -v buggy state.txt"

	listener := &mockHealListener{}
	coord := NewAutoHealCoordinator(tempDir, listener)

	fixApplied := false
	session, err := coord.ExecuteHealingLoop(
		context.Background(),
		"Fix buggy state file",
		reproCmd,
		func(ctx context.Context) error {
			// Apply fix by writing "fixed"
			fixApplied = true
			return os.WriteFile(targetFile, []byte("fixed"), 0644)
		},
	)

	if err != nil {
		t.Fatalf("ExecuteHealingLoop() error = %v", err)
	}
	if !fixApplied {
		t.Fatal("expected fix function to be executed")
	}
	if session.State != StateVerified {
		t.Fatalf("expected session state %s, got %s", StateVerified, session.State)
	}
	if len(listener.steps) < 4 {
		t.Fatalf("expected at least 4 healing steps, got %d", len(listener.steps))
	}
}
