package accounts

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRouterReportSuccessClearsCooldown(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatal(err)
	}
	account := &Account{ID: "a", Provider: ProviderTypeCodex, AccessToken: "token", CooldownUntil: time.Now().Add(time.Hour), LastError: "rate limited"}
	if err := store.AddOrUpdate(account); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(store)
	router.ReportSuccess(account.ID)
	updated := store.Accounts[account.ID]
	if !updated.CooldownUntil.IsZero() || updated.LastError != "" {
		t.Fatalf("success did not reset account: %+v", updated)
	}
}
