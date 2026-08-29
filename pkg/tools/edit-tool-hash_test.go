package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestEditToolHashAnchoredStaleRejection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.go")
	original := "package main\n\nfunc main() {\n\tprintln(\"v1\")\n}\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	edit := &EditTool{BaseDir: dir}
	staleHash := strings.Repeat("a", 64)
	_, err := edit.Execute(t.Context(), map[string]interface{}{
		"TargetFile": path, "FileHash": staleHash, "TargetContent": "v1", "ReplacementContent": "v2",
	})
	if err == nil || !strings.Contains(err.Error(), "stale edit rejected") {
		t.Fatalf("expected stale rejection, got %v", err)
	}

	current := fileFingerprint([]byte(original))
	out, err := edit.Execute(t.Context(), map[string]interface{}{
		"TargetFile": path, "FileHash": current, "TargetContent": "v1", "ReplacementContent": "v2",
	})
	if err != nil {
		t.Fatalf("fresh-hash edit failed: %v", err)
	}
	if !strings.Contains(out, "Successfully replaced") || !strings.Contains(out, "FileHash:") {
		t.Fatalf("unexpected output: %s", out)
	}
	updated, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(updated), "v2") {
		t.Fatalf("file not updated: %v %s", err, string(updated))
	}
}

func TestEditToolConcurrentHashEditsAllowOneWinner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.txt")
	original := []byte("value=old\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := fileFingerprint(original)
	tool := &EditTool{BaseDir: dir}
	args := func(value string) map[string]interface{} {
		return map[string]interface{}{"TargetFile": path, "FileHash": hash, "TargetContent": "old", "ReplacementContent": value}
	}
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, value := range []string{"one", "two"} {
		group.Add(1)
		go func(value string) {
			defer group.Done()
			_, err := tool.Execute(context.Background(), args(value))
			results <- err
		}(value)
	}
	group.Wait()
	close(results)
	wins, stale := 0, 0
	for err := range results {
		if err == nil {
			wins++
		} else if strings.Contains(err.Error(), "stale edit rejected") {
			stale++
		}
	}
	if wins != 1 || stale != 1 {
		t.Fatalf("wins=%d stale=%d", wins, stale)
	}
}

func TestEditToolHashOptionalBackwardCompatible(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o600); err != nil {
		t.Fatal(err)
	}
	edit := &EditTool{BaseDir: dir}
	if _, err := edit.Execute(t.Context(), map[string]interface{}{
		"TargetFile": path, "TargetContent": "hello", "ReplacementContent": "hi",
	}); err != nil {
		t.Fatalf("edit without FileHash should still work: %v", err)
	}
}
