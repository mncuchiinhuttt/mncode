package ui

import (
	"path/filepath"
	"testing"

	"mncode/pkg/agent"
	"mncode/pkg/config"
	"mncode/pkg/provider"
	"mncode/pkg/stats"
)

func TestValidateTelemetryEndpointRequiresHTTPS(t *testing.T) {
	for _, endpoint := range []string{
		"http://telemetry.example.test/sync",
		"https://user:pass@telemetry.example.test/sync",
		"https://telemetry.example.test/sync#fragment",
		"https:///missing-host",
	} {
		if err := ValidateTelemetryEndpoint(endpoint); err == nil {
			t.Errorf("expected endpoint to be rejected: %q", endpoint)
		}
	}
	if err := ValidateTelemetryEndpoint("https://telemetry.example.test/sync"); err != nil {
		t.Fatalf("valid HTTPS endpoint rejected: %v", err)
	}
}
func TestBuildTelemetryPayloadUsesRecordedValues(t *testing.T) {
	tracker := stats.NewTrackerAt(filepath.Join(t.TempDir(), "usage.json"))
	// RecordWithThinking also verifies the same aggregation used by sync.
	if err := tracker.RecordWithThinking("model-a", "account-a", 10, 20, 30); err != nil {
		t.Fatalf("record usage: %v", err)
	}
	session := &agent.Session{
		Config: &config.Config{},
		History: []provider.Message{{
			Role:      provider.RoleAssistant,
			ToolCalls: []provider.ToolCall{{Name: "run_command"}, {Name: "run_command"}, {Name: "read_file"}},
		}},
	}
	payload := buildTelemetryPayload(session, tracker, "today")
	if payload.SessionCount != 1 || payload.Models["model-a"] != 1 ||
		payload.InputTokens != 10 || payload.OutputTokens != 20 ||
		payload.ThinkingTokens != 30 || payload.TotalTokens != 60 {
		t.Fatalf("payload did not use tracker records: %+v", payload)
	}
	if payload.Tools["run_command"] != 2 || payload.Tools["read_file"] != 1 {
		t.Fatalf("payload did not count real tool calls: %+v", payload.Tools)
	}
}

func TestTelemetrySyncRequiresExplicitConsent(t *testing.T) {
	cfg := &config.Config{}
	if telemetrySyncConsented(cfg) {
		t.Fatal("empty consent setting must not grant consent")
	}
	cfg.SetSetting("telemetry_sync_consent", "true")
	if !telemetrySyncConsented(cfg) {
		t.Fatal("explicit consent setting should grant consent")
	}
}
