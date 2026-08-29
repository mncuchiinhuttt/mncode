package hooks

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSilentAutoFormatGo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-hooks-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Unformatted Go code (bad spacing)
	unformatted := "package main\nfunc  Foo(   a int ) int {\nreturn    a + 1\n}\n"
	target := filepath.Join(tempDir, "foo.go")
	if err := os.WriteFile(target, []byte(unformatted), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SilentAutoFormat(context.Background(), target); err != nil {
		t.Fatalf("SilentAutoFormat() error = %v", err)
	}

	formatted, _ := os.ReadFile(target)
	if strings.Contains(string(formatted), "func  Foo") {
		t.Fatalf("file was not auto-formatted by gofmt:\n%s", string(formatted))
	}
}

func TestRunCustomHook(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-hook-run-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	hooksDir := filepath.Join(tempDir, ".mncode", "hooks")
	if err := os.MkdirAll(hooksDir, 0700); err != nil {
		t.Fatal(err)
	}

	logFile := filepath.Join(tempDir, "hook-executed.log")
	hookScript := filepath.Join(hooksDir, "post_edit.sh")
	scriptContent := "#!/bin/sh\necho \"Hook triggered for $1\" > \"" + logFile + "\"\n"
	if err := os.WriteFile(hookScript, []byte(scriptContent), 0755); err != nil {
		t.Fatal(err)
	}

	targetFile := filepath.Join(tempDir, "target.txt")
	if err := RunHook(context.Background(), tempDir, EventPostEdit, targetFile); err != nil {
		t.Fatalf("RunHook() error = %v", err)
	}

	logData, err := os.ReadFile(logFile)
	if err != nil || !strings.Contains(string(logData), "Hook triggered for") {
		t.Fatalf("expected custom hook log, got err=%v data=%q", err, string(logData))
	}
}
