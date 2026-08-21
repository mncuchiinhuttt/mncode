package stats

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTokenTracker(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mncode-stats-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tracker := &Tracker{
		filePath: filepath.Join(tmpDir, "usage.json"),
		store: UsageStore{
			Daily:    make(map[string]*UsageSummary),
			Monthly:  make(map[string]*UsageSummary),
			ByModel:  make(map[string]*UsageSummary),
			Lifetime: &UsageSummary{},
			History:  make([]TokenRecord, 0),
		},
	}

	// 1. Record first usage
	tracker.Record("gemini-2.5-pro", "acc-1", 1000, 500)

	today := tracker.GetToday()
	if today.InputTokens != 1000 || today.OutputTokens != 500 || today.TotalTokens != 1500 || today.Requests != 1 {
		t.Errorf("unexpected today stats: %+v", today)
	}

	month := tracker.GetMonth()
	if month.TotalTokens != 1500 || month.Requests != 1 {
		t.Errorf("unexpected month stats: %+v", month)
	}

	lifetime := tracker.GetLifetime()
	if lifetime.TotalTokens != 1500 || lifetime.Requests != 1 {
		t.Errorf("unexpected lifetime stats: %+v", lifetime)
	}

	// 2. Record second usage
	tracker.Record("gemini-2.5-pro", "acc-1", 200, 300)
	today = tracker.GetToday()
	if today.TotalTokens != 2000 || today.Requests != 2 {
		t.Errorf("expected 2000 total tokens today, got %d", today.TotalTokens)
	}

	// 3. Test reload from disk
	tracker2 := &Tracker{
		filePath: filepath.Join(tmpDir, "usage.json"),
		store: UsageStore{
			Daily:    make(map[string]*UsageSummary),
			Monthly:  make(map[string]*UsageSummary),
			ByModel:  make(map[string]*UsageSummary),
			Lifetime: &UsageSummary{},
			History:  make([]TokenRecord, 0),
		},
	}
	tracker2.load()
	if tracker2.GetLifetime().TotalTokens != 2000 {
		t.Errorf("expected 2000 reloaded lifetime tokens, got %d", tracker2.GetLifetime().TotalTokens)
	}
}
