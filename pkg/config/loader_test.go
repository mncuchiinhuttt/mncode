package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Provider != ProviderAnthropic {
		t.Errorf("expected provider %s, got %s", ProviderAnthropic, cfg.Provider)
	}
	if cfg.ThinkingBudget != 8192 {
		t.Errorf("expected thinking budget 8192, got %d", cfg.ThinkingBudget)
	}
	if cfg.Effort != "high" {
		t.Errorf("expected effort 'high', got %s", cfg.Effort)
	}
}

func TestLoadDotEnv(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	content := "TEST_KEY_MNCODE=mncode_test_value\n# Comment\nANOTHER_KEY=\"quoted_value\"\n"

	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test .env: %v", err)
	}

	if err := LoadDotEnv(envPath); err != nil {
		t.Fatalf("LoadDotEnv failed: %v", err)
	}

	if val := os.Getenv("TEST_KEY_MNCODE"); val != "mncode_test_value" {
		t.Errorf("expected TEST_KEY_MNCODE to be 'mncode_test_value', got '%s'", val)
	}
	if val := os.Getenv("ANOTHER_KEY"); val != "quoted_value" {
		t.Errorf("expected ANOTHER_KEY to be 'quoted_value', got '%s'", val)
	}
}
