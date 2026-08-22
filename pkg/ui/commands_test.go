package ui

import (
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"testing"
)

func TestSemanticCommitMessage(t *testing.T) {
	status := " M pkg/ui/diff-commands.go\n M pkg/ui/commands_test.go\n"
	msg := generateSemanticCommitMessage(status, ".")
	if msg == "" {
		t.Fatalf("expected non-empty commit message")
	}
	if !testing.Short() && len(msg) < 10 {
		t.Errorf("commit message too short: %s", msg)
	}
}

func TestDoctorCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	s := &agent.Session{
		ID:           "test-session",
		WorkspaceDir: ".",
		Config:       cfg,
	}
	HandleDoctorCommand([]string{"/doctor"}, s)
}

func TestDiffCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	s := &agent.Session{
		ID:           "test-session",
		WorkspaceDir: ".",
		Config:       cfg,
	}
	HandleDiffCommand([]string{"/diff"}, s)
}

func TestReviewCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	s := &agent.Session{
		ID:           "test-session",
		WorkspaceDir: ".",
		Config:       cfg,
	}
	HandleReviewCommand([]string{"/review"}, s)
}

func TestScratchCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	s := &agent.Session{
		ID:           "test-session",
		WorkspaceDir: ".",
		Config:       cfg,
	}
	HandleScratchCommand([]string{"/scratch", "go"}, s)
	HandleScratchCommand([]string{"/scratch", "view"}, s)
}

func TestTreeCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	s := &agent.Session{
		ID:           "test-session",
		WorkspaceDir: ".",
		Config:       cfg,
	}
	HandleTreeCommand([]string{"/tree", "1"}, s)
}

func TestResolveCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	s := &agent.Session{
		ID:           "test-session",
		WorkspaceDir: ".",
		Config:       cfg,
	}
	HandleResolveCommand([]string{"/resolve"}, s)
}

func TestDBCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	s := &agent.Session{
		ID:           "test-session",
		WorkspaceDir: ".",
		Config:       cfg,
	}
	HandleDBCommand([]string{"/db"}, s)
}

func TestAPICommand(t *testing.T) {
	HandleAPICommand([]string{"/api"})
}

func TestChangelogCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	s := &agent.Session{
		ID:           "test-session",
		WorkspaceDir: ".",
		Config:       cfg,
	}
	HandleChangelogCommand([]string{"/changelog"}, s)
}
