package accounts

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreAndRouter(t *testing.T) {
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "accounts.json")

	store, err := NewStore(storeFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// 1. Add 2 Antigravity accounts
	acc1 := &Account{
		ID:          "antigravity-1",
		Email:       "user1@gmail.com",
		Provider:    ProviderTypeAntigravity,
		AccessToken: "token_anti_1",
		IsActive:    false,
	}
	acc2 := &Account{
		ID:          "antigravity-2",
		Email:       "user2@gmail.com",
		Provider:    ProviderTypeAntigravity,
		AccessToken: "token_anti_2",
		IsActive:    false,
	}

	_ = store.AddOrUpdate(acc1)
	_ = store.AddOrUpdate(acc2)

	router := NewRouter(store)

	// 2. Test Round Robin
	first, err := router.GetNextAccount(ProviderTypeAntigravity)
	if err != nil || first == nil {
		t.Fatalf("expected account, got error: %v", err)
	}

	second, err := router.GetNextAccount(ProviderTypeAntigravity)
	if err != nil || second == nil {
		t.Fatalf("expected second account, got error: %v", err)
	}

	if first.ID == second.ID {
		t.Errorf("expected different accounts in round robin, got same ID: %s", first.ID)
	}

	// 3. Test Cooldown on 429 rate limit
	router.ReportFailure(first.ID, 429, "Too Many Requests")

	// Next rotation must skip first.ID and pick second.ID
	picked, err := router.GetNextAccount(ProviderTypeAntigravity)
	if err != nil {
		t.Fatalf("expected available account after cooldown: %v", err)
	}
	if picked.ID == first.ID {
		t.Errorf("expected to skip account on cooldown, but got %s", picked.ID)
	}
}

func TestCodexAccountManagement(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(filepath.Join(tmpDir, "accounts.json"))

	acc, err := AddCodexAccount(store, "work@company.com", "sk-test-openai-token")
	if err != nil {
		t.Fatalf("failed to add codex account: %v", err)
	}

	if acc.Provider != ProviderTypeCodex {
		t.Errorf("expected provider codex, got %s", acc.Provider)
	}

	accounts := store.ListByProvider(ProviderTypeCodex)
	if len(accounts) != 1 {
		t.Errorf("expected 1 codex account, got %d", len(accounts))
	}
}

func TestParseCredentialExpiry(t *testing.T) {
	if got := parseCredentialExpiry("1700000000"); got.Unix() != 1700000000 {
		t.Fatalf("numeric expiry = %v", got)
	}
	want := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	if got := parseCredentialExpiry(want.Format(time.RFC3339)); !got.Equal(want) {
		t.Fatalf("RFC3339 expiry = %v, want %v", got, want)
	}
}
