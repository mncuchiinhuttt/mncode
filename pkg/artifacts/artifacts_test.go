package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretScrubber(t *testing.T) {
	input := `
Anthropic: sk-ant-api03-abcdef1234567890abcdef1234567890
OpenAI: sk-proj-1234567890abcdef1234567890abcdef12345678
	Google: ya29.a0AfH6SM1234567890abcdef1234567890
Gemini: AIzaSy1234567890abcdef1234567890
Key:
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA0...
-----END RSA PRIVATE KEY-----
`
	scrubbed := ScrubSecrets(input)

	if strings.Contains(scrubbed, "sk-ant-api03") || strings.Contains(scrubbed, "sk-proj-") || strings.Contains(scrubbed, "ya29.") || strings.Contains(scrubbed, "ghp_") || strings.Contains(scrubbed, "AIzaSy") {
		t.Fatalf("scrubber failed to redact secrets:\n%s", scrubbed)
	}
	if !strings.Contains(scrubbed, "[REDACTED_ANTHROPIC_KEY]") {
		t.Errorf("missing anthropic redaction marker")
	}
	if !strings.Contains(scrubbed, "[REDACTED_PRIVATE_KEY_BLOCK]") {
		t.Errorf("missing private key block redaction marker")
	}
}

func TestArtifactStoreAndResolver(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-artifacts-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10"
	id, err := store.Save(content)
	if err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}

	retrieved, err := store.Get(id)
	if err != nil {
		t.Fatalf("store.Get() error = %v", err)
	}
	if retrieved != content {
		t.Fatalf("retrieved = %q, want %q", retrieved, content)
	}

	// Verify 0600 file permissions
	filePath := filepath.Join(tempDir, id+".txt")
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestTruncatorGeneratesArtifact(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-truncator-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	var longText strings.Builder
	for i := 1; i <= 200; i++ {
		longText.WriteString("This is a log output line from a test runner\n")
	}

	truncated := TruncateOutput(longText.String(), store)
	if !strings.Contains(truncated, "artifact://") {
		t.Fatalf("expected artifact:// recovery URI in truncated output:\n%s", truncated)
	}
	if !strings.Contains(truncated, "lines elided") {
		t.Fatalf("expected elided notice in truncated output:\n%s", truncated)
	}
}

func TestSliceContent(t *testing.T) {
	content := "One\nTwo\nThree\nFour\nFive"
	sliced, _ := sliceContent(content, "2-4")
	expected := "Two\nThree\nFour"
	if sliced != expected {
		t.Errorf("sliceContent(2-4) = %q, want %q", sliced, expected)
	}
}
func TestVirtualURIsRejectWorkspaceEscape(t *testing.T) {
	parent := t.TempDir()
	workspace := filepath.Join(parent, "workspace")
	if err := os.MkdirAll(filepath.Join(workspace, ".mncode", "scratchpad"), 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.txt")
	if err := os.WriteFile(outside, []byte("sensitive"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadVirtualURI("local://../outside.txt", workspace); err == nil {
		t.Fatal("expected local URI traversal to be rejected")
	}
	if _, err := ReadVirtualURI("artifact://../../outside", workspace); err == nil {
		t.Fatal("expected artifact URI traversal to be rejected")
	}
}

func TestVirtualURIReadsOnlyRegularScratchpadFiles(t *testing.T) {
	workspace := t.TempDir()
	scratchpad := filepath.Join(workspace, ".mncode", "scratchpad")
	if err := os.MkdirAll(scratchpad, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scratchpad, "note.txt"), []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	content, err := ReadVirtualURI("local://note.txt", workspace)
	if err != nil || content != "safe" {
		t.Fatalf("expected scratchpad read, content=%q err=%v", content, err)
	}
}
