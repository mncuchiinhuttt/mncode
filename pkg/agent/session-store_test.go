package agent

import (
	"os"
	"path/filepath"
	"testing"

	"mncode/pkg/config"
	"mncode/pkg/provider"
)

func TestLoadSavedSessionRejectsPathTraversal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	outside := filepath.Join(home, ".mncode", "outside.json")
	if err := os.MkdirAll(filepath.Dir(outside), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte(`{"id":"outside","messages":[{"role":"user","content":"secret"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadSavedSession("../outside"); err == nil {
		t.Fatal("expected traversal session ID to be rejected")
	}
	if _, err := LoadSavedSession("..\\outside"); err == nil {
		t.Fatal("expected backslash traversal session ID to be rejected")
	}
}

func TestSaveSessionRestrictsHistoryPermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	session := &Session{
		ID:      "safe-session",
		Config:  &config.Config{Model: "test-model"},
		History: []provider.Message{{Role: provider.RoleUser, Content: "private prompt"}},
	}
	if err := session.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	dir := GetSessionsDir()
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat sessions directory: %v", err)
	}
	if got, want := dirInfo.Mode().Perm(), os.FileMode(0o700); got != want {
		t.Fatalf("sessions directory mode = %04o, want %04o", got, want)
	}
	fileInfo, err := os.Stat(filepath.Join(dir, "safe-session.json"))
	if err != nil {
		t.Fatalf("stat session file: %v", err)
	}
	if got, want := fileInfo.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("session file mode = %04o, want %04o", got, want)
	}
}

func TestLoadSavedSessionPrefersCanonicalState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	session := &Session{
		ID:      "canonical-session",
		Config:  &config.Config{Model: "canonical-model"},
		History: []provider.Message{{Role: provider.RoleUser, Content: "canonical prompt"}},
	}
	if err := session.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	legacyPath := filepath.Join(GetSessionsDir(), "canonical-session.json")
	if err := os.WriteFile(legacyPath, []byte(`{"id":"canonical-session","messages":[{"role":"user","content":"stale"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSavedSession("canonical-session")
	if err != nil {
		t.Fatalf("LoadSavedSession() error = %v", err)
	}
	if got := loaded.Messages[0].Content; got != "canonical prompt" {
		t.Fatalf("loaded content = %q, want canonical state", got)
	}
}
