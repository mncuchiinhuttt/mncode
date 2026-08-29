package stats

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStreakStatsDoesNotFabricateValues(t *testing.T) {
	tracker := &Tracker{
		filePath: filepath.Join(t.TempDir(), "usage.json"),
		store: UsageStore{
			Daily: make(map[string]*UsageSummary), Monthly: make(map[string]*UsageSummary),
			ByModel: make(map[string]*UsageSummary), Lifetime: &UsageSummary{},
		},
	}
	info := tracker.GetStreakStats()
	if info.Sessions != 0 || info.TotalDays != 0 || info.ActiveDays != 0 ||
		info.LongestSession != "" || info.FavoriteModel != "" || info.MostActiveDay != "" {
		t.Fatalf("empty tracker reported fabricated values: %+v", info)
	}
}

func TestStreakStatsUsesRecordedDaysAndThinkingTokens(t *testing.T) {
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tracker := &Tracker{
		filePath: filepath.Join(t.TempDir(), "usage.json"),
		store: UsageStore{
			Daily: map[string]*UsageSummary{
				today: {TotalTokens: 60}, yesterday: {TotalTokens: 10},
			},
			Monthly:  make(map[string]*UsageSummary),
			ByModel:  map[string]*UsageSummary{"model-a": {TotalTokens: 60}},
			Lifetime: &UsageSummary{TotalTokens: 70, InputTokens: 20, OutputTokens: 30, ThinkingTokens: 20},
			History:  []TokenRecord{{Timestamp: now, TotalTokens: 60}, {Timestamp: now, TotalTokens: 10}},
		},
	}
	info := tracker.GetStreakStats()
	if info.Sessions != 2 || info.ActiveDays != 2 || info.TotalDays != 2 ||
		info.LongestStreak != 2 || info.ThinkingTokens != 20 ||
		info.FavoriteModel != "model-a" {
		t.Fatalf("unexpected recorded streak stats: %+v", info)
	}
}
