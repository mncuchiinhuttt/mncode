package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveConfigRestrictsCredentialFilePermissions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := SaveConfig(&Config{
		APIKey:         "api-key-fixture",
		OpenCodeAPIKey: "opencode-key-fixture",
		TelemetryKey:   "telemetry-key-fixture",
	}); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	path, err := GetConfigFilePath()
	if err != nil {
		t.Fatalf("GetConfigFilePath() error = %v", err)
	}
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("config file permissions = %04o, want %04o", got, want)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat config directory: %v", err)
	}
	if got, want := dirInfo.Mode().Perm(), os.FileMode(0o700); got != want {
		t.Fatalf("config directory permissions = %04o, want %04o", got, want)
	}
}
