package agent

import (
	"mncode/pkg/config"
	"mncode/pkg/memory"
	"strings"
	"testing"
)

func TestPromptTiersAssembly(t *testing.T) {
	tiers := PromptTiers{
		Stable:   "<identity>mncode</identity>",
		Context:  "<rules>test rules</rules>",
		Volatile: "<user_information>test user</user_information>",
	}

	assembled := tiers.Assemble()
	if !strings.Contains(assembled, "<identity>mncode</identity>") {
		t.Errorf("assembled prompt missing stable tier")
	}
	if !strings.Contains(assembled, "<rules>test rules</rules>") {
		t.Errorf("assembled prompt missing context tier")
	}
	if !strings.Contains(assembled, "<user_information>test user</user_information>") {
		t.Errorf("assembled prompt missing volatile tier")
	}

	// Verify order: Stable -> Context -> Volatile
	posStable := strings.Index(assembled, "<identity>")
	posContext := strings.Index(assembled, "<rules>")
	posVolatile := strings.Index(assembled, "<user_information>")

	if posStable > posContext || posContext > posVolatile {
		t.Errorf("tiers assembled in wrong order: stable=%d, context=%d, volatile=%d", posStable, posContext, posVolatile)
	}
}

func TestBuildPromptTiers(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CodingLevel = 2
	cfg.Effort = "high"

	session := &Session{
		WorkspaceDir: "/tmp/test-workspace",
		Config:       cfg,
	}

	snapshot := memory.MemorySnapshot{
		Version: "v1",
		Entries: []memory.Entry{
			{Text: "Test memory fact"},
		},
	}
	tiers := session.BuildPromptTiers(snapshot)

	if !strings.Contains(tiers.Stable, "<identity>") {
		t.Errorf("stable tier missing identity")
	}
	if !strings.Contains(tiers.Volatile, "Coding Level: 2") {
		t.Errorf("volatile tier missing coding level")
	}

	fullPrompt := session.BuildSystemPrompt()
	if !strings.Contains(fullPrompt, "<identity>") || !strings.Contains(fullPrompt, "/tmp/test-workspace") {
		t.Errorf("BuildSystemPrompt missing expected content")
	}

	// Test cache invalidation
	session.InvalidatePromptCache()
}
