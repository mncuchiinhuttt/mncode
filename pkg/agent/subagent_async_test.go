package agent

import (
	"context"
	"mncode/pkg/config"
	"mncode/pkg/orchestration"
	"mncode/pkg/skills"
	"testing"
	"time"
)

func TestSubagentRunnerAsync(t *testing.T) {
	catalog := &skills.Catalog{
		Agents: map[string]*skills.Agent{
			"scout": {
				Name:   "scout",
				Role:   "Codebase Scout",
				Prompt: "You are scout.",
			},
		},
	}

	session := &Session{
		ID:           "test-session-123",
		WorkspaceDir: t.TempDir(),
		Config:       config.DefaultConfig(),
		Catalog:      catalog,
	}

	runner := &SubagentRunner{ParentSession: session}
	mgr := orchestration.NewRunManager(nil)

	var events []orchestration.EventEnvelope
	mgr.Subscribe(func(e orchestration.EventEnvelope) {
		events = append(events, e)
	})

	run, err := runner.RunAsync(context.Background(), "scout", "find all go files", mgr)
	if err != nil {
		t.Fatalf("RunAsync failed: %v", err)
	}

	if run.ID() == "" {
		t.Fatal("expected non-empty run ID")
	}

	snap, err := mgr.Wait(context.Background(), run.ID(), 2*time.Second)
	if err != nil {
		t.Fatalf("wait error: %v", err)
	}

	if snap.Meta.ChatID != "test-session-123" {
		t.Fatalf("expected ChatID test-session-123, got %s", snap.Meta.ChatID)
	}
}
