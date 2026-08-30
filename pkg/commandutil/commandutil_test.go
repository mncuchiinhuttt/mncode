package commandutil

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveWorkspaceRejectsEscape(t *testing.T) {
	root := t.TempDir()
	workspace, err := ResolveWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Relative(filepath.Join(workspace.Root, "inside.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Relative(filepath.Join(root, "..", "outside")); err == nil {
		t.Fatal("expected relative escape rejection")
	}
}

func TestRunBoundedCapsOutputAndTimeout(t *testing.T) {
	var direct cappedBuffer
	direct.limit = 4
	_, _ = direct.Write([]byte("123456"))
	if string(direct.Bytes()) != "1234" || !direct.truncated {
		t.Fatalf("direct cap failed: %q truncated=%t", direct.Bytes(), direct.truncated)
	}
	var commandBuffer cappedBuffer
	commandBuffer.limit = 4
	cmd := exec.Command("printf", "123456")
	cmd.Stdout = &commandBuffer
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if string(commandBuffer.Bytes()) != "1234" || !commandBuffer.truncated {
		t.Fatalf("exec cap failed: %q truncated=%t", commandBuffer.Bytes(), commandBuffer.truncated)
	}
	root := t.TempDir()
	limits := Limits{Timeout: time.Second, MaxOutputBytes: 4, MaxFiles: 1, MaxFileBytes: 1024}
	stdout, _, err := RunBounded(context.Background(), root, []string{"printf", "123456"}, limits)
	if !errors.Is(err, ErrOutputLimit) || string(stdout) != "1234" {
		t.Fatalf("unexpected cap result %q %v", stdout, err)
	}
	_, _, err = RunBounded(context.Background(), root, []string{"sleep", "2"}, Limits{Timeout: 10 * time.Millisecond, MaxOutputBytes: 10})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected timeout, got %v", err)
	}
	_, _, err = RunBounded(context.Background(), root, []string{"sh", "-c", "sleep 2"}, Limits{Timeout: 10 * time.Millisecond, MaxOutputBytes: 10})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected descendant timeout, got %v", err)
	}
}

func TestWritePrivateJSONPermissions(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "record.json")
	if err := WritePrivateJSON(path, map[string]string{"value": "sk-proj-12345678901234567890123456789012"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode %o", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" || string(data) == "sk-proj-12345678901234567890123456789012" {
		t.Fatal("secret not scrubbed")
	}
}
