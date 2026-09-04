package ui

import (
	"testing"
)

func TestPickDebateModel(t *testing.T) {
	cases := []struct {
		input    string
		fallback string
		want     string
	}{
		{"", "claude-sonnet-4-6", "claude-sonnet-4-6"},
		{"1", "claude-sonnet-4-6", "claude-sonnet-4-6"},
		{"2", "claude-sonnet-4-6", "deepseek-reasoner"},
		{"o3", "claude-sonnet-4-6", "o3"},
		{"custom-finetuned", "claude-sonnet-4-6", "custom-finetuned"},
	}

	for _, c := range cases {
		got := pickDebateModel(c.input, c.fallback)
		if got != c.want {
			t.Errorf("pickDebateModel(%q, %q) = %q, want %q", c.input, c.fallback, got, c.want)
		}
	}
}

func TestTruncateStr(t *testing.T) {
	if got := truncateStr("short", 10); got != "short" {
		t.Errorf("got %q, want short", got)
	}
	if got := truncateStr("super-long-model-identifier-string", 12); got != "super-lon..." {
		t.Errorf("got %q, want super-lon...", got)
	}
}
func TestSelectDebateModelsNonTerminalFallback(t *testing.T) {
	models := selectDebateModelsInteractive()
	if len(models) < 2 {
		t.Fatalf("expected at least 2 default models, got %v", models)
	}
	if models[0] != "claude-sonnet-4-6" || models[1] != "deepseek-reasoner" {
		t.Fatalf("unexpected fallback models: %v", models)
	}
}
