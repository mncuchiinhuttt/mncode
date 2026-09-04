package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)
func TestHierarchicalMemoryStoreAndCrossSessionSharing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-memory-ws-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	glPath := filepath.Join(tempDir, "global.json")
	wsPath := filepath.Join(tempDir, "workspace.json")

	// Session A writes to shared workspace memory
	storeA := NewHierarchicalStoreWithPaths(glPath, wsPath)

	item := MemoryItem{
		Topic:      "auth-header",
		Category:   CategoryGotchaBug,
		Tier:       TierWorkspace,
		Summary:    "Use Bearer prefix on API v2",
		Correction: "Header must be Authorization: Bearer <token>",
	}
	if err := storeA.Save(item); err != nil {
		t.Fatalf("storeA.Save() error = %v", err)
	}

	// Session B in the same workspace opens and immediately retrieves Session A's memory
	storeB := NewHierarchicalStoreWithPaths(glPath, wsPath)

	allB := storeB.ListAll()
	if len(allB) != 1 {
		t.Fatalf("expected Session B to find 1 shared memory, got %d", len(allB))
	}
	if allB[0].Topic != "auth-header" || allB[0].Tier != TierWorkspace {
		t.Fatalf("unexpected memory data in Session B: %+v", allB[0])
	}
}
func TestHermesReflectionAndSuperseding(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-memory-evolve-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	glPath := filepath.Join(tempDir, "global.json")
	wsPath := filepath.Join(tempDir, "workspace.json")
	store := NewHierarchicalStoreWithPaths(glPath, wsPath)

	// 1. Initial lesson
	lesson1 := ReflectOnFailure("go_test", "go test ./...", "exit code 1: test timed out after 30s", "pass -timeout=60s flag")
	item1, updated1, err := EvolveMemory(store, lesson1, TierWorkspace)
	if err != nil || updated1 {
		t.Fatalf("initial evolve err=%v, updated=%v", err, updated1)
	}
	if item1.ID == "" {
		t.Fatal("expected item1 ID to be set")
	}

	// 2. Refined lesson on same topic (supersedes initial lesson)
	lesson2 := ReflectiveLesson{
		Topic:      lesson1.Topic,
		Category:   CategoryGotchaBug,
		Summary:    "Pass -timeout=120s flag for database integration tests",
		Correction: "use -timeout=120s",
		Confidence: 5,
		Source:     "auto-reflection",
	}
	item2, updated2, err := EvolveMemory(store, lesson2, TierWorkspace)
	if err != nil || !updated2 {
		t.Fatalf("evolved lesson err=%v, updated=%v", err, updated2)
	}
	if item2.SupersedesID != item1.ID {
		t.Fatalf("expected item2 SupersedesID=%q, got %q", item1.ID, item2.SupersedesID)
	}

	// Store should have only 1 active memory item on this topic
	all := store.ListAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 active evolved item, got %d", len(all))
	}
	if all[0].Summary != lesson2.Summary {
		t.Fatalf("expected updated summary %q, got %q", lesson2.Summary, all[0].Summary)
	}
}
func TestGetRelevantMemoriesAndPromptInjection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-memory-retrieval-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	glPath := filepath.Join(tempDir, "global.json")
	wsPath := filepath.Join(tempDir, "workspace.json")
	store := NewHierarchicalStoreWithPaths(glPath, wsPath)
	_ = store.Save(MemoryItem{Topic: "payment-webhook", Category: CategoryArchitecture, Tier: TierWorkspace, Summary: "Verify HMAC SHA256 signatures on webhooks"})
	_ = store.Save(MemoryItem{Topic: "language-pref", Category: CategoryUserPreference, Tier: TierGlobal, Summary: "Reply in Vietnamese"})

	wsItems, glItems := GetRelevantMemories(store, "How to handle payment-webhook in our API?", []string{"pkg/api/webhook.go"}, 5)
	if len(wsItems) != 1 || wsItems[0].Topic != "payment-webhook" {
		t.Fatalf("expected workspace item match, got %v", wsItems)
	}

	xml := FormatPromptMemoryContext(wsItems, glItems)
	if !strings.Contains(xml, "<shared-workspace-memories>") || !strings.Contains(xml, "payment-webhook") {
		t.Fatalf("unexpected prompt XML:\n%s", xml)
	}
}
