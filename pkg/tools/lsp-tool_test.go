package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// lookGopls finds gopls on PATH or ~/go/bin.
func lookGopls() (string, error) {
	if p, err := exec.LookPath("gopls"); err == nil {
		return p, nil
	}
	home, _ := os.UserHomeDir()
	return exec.LookPath(filepath.Join(home, "go", "bin", "gopls"))
}

// requiresGopls skips the test when gopls is not on PATH.
func requiresGopls(t *testing.T) {
	t.Helper()
	if _, err := lookGopls(); err != nil {
		t.Skip("gopls not installed; skipping LSP integration test")
	}
}

func TestLSPToolDefinition(t *testing.T) {
	requiresGopls(t)
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/lsptest\n\ngo 1.24\n")
	writeFile(t, dir, "main.go", "package main\n\nimport \"fmt\"\n\nfunc greet() string {\n\treturn \"hi\"\n}\n\nfunc main() {\n\tfmt.Println(greet())\n}\n")

	l := &LSPTool{BaseDir: dir}
	out, err := l.Execute(context.Background(), map[string]interface{}{
		"action": "definition",
		"file":   "main.go",
		"line":   10,
		"column": 15, // inside greet() call
	})
	if err != nil {
		t.Fatalf("definition failed: %v", err)
	}
	if !strings.Contains(out, "main.go") {
		t.Fatalf("expected a location in main.go, got: %s", out)
	}
}

func TestLSPToolReferences(t *testing.T) {
	requiresGopls(t)
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/lsptest\n\ngo 1.24\n")
	writeFile(t, dir, "main.go", "package main\n\nimport \"fmt\"\n\nfunc greet() string {\n\treturn \"hi\"\n}\n\nfunc main() {\n\tfmt.Println(greet())\n\tfmt.Println(greet())\n}\n")

	l := &LSPTool{BaseDir: dir}
	out, err := l.Execute(context.Background(), map[string]interface{}{
		"action": "references",
		"file":   "main.go",
		"line":   5,
		"column": 6, // on "greet" declaration
	})
	if err != nil {
		t.Fatalf("references failed: %v", err)
	}
	if !strings.Contains(out, "Found") || !strings.Contains(out, "main.go") {
		t.Fatalf("unexpected references output: %s", out)
	}
}

func TestLSPToolRejectsUnsupportedLanguage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notes.py", "x = 1\n")
	l := &LSPTool{BaseDir: dir}
	_, err := l.Execute(context.Background(), map[string]interface{}{
		"action": "definition", "file": "notes.py", "line": 1, "column": 1,
	})
	if err == nil || !strings.Contains(err.Error(), "no language server") {
		t.Fatalf("expected unsupported-language error, got %v", err)
	}
}

func TestApplyWorkspaceEditHandlesNullResult(t *testing.T) {
	result, err := applyWorkspaceEdit(nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if result != "Rename produced no edits." {
		t.Fatalf("result = %q", result)
	}
}

func TestApplyWorkspaceEditCanonicalizesSymlinkWorkspace(t *testing.T) {
	root := t.TempDir()
	realRoot := filepath.Join(root, "real")
	if err := os.Mkdir(realRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	linkRoot := filepath.Join(root, "link")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	path := filepath.Join(realRoot, "main.go")
	writeFile(t, realRoot, "main.go", "package main\n")
	raw, _ := json.Marshal(lspWorkspaceEdit{Changes: map[string][]lspTextEdit{
		pathToURI(path): {{Range: lspRange{Start: lspPosition{Line: 0, Character: 12}, End: lspPosition{Line: 0, Character: 12}}, NewText: "\n// renamed"}},
	}})
	if _, err := applyWorkspaceEdit(raw, linkRoot); err != nil {
		t.Fatalf("symlink workspace rename failed: %v", err)
	}
}

func TestApplyWorkspaceEditPreflightsAllFilesBeforeCommit(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.go")
	writeFile(t, root, "first.go", "package main\n")
	missing := filepath.Join(root, "missing.go")
	raw, _ := json.Marshal(lspWorkspaceEdit{Changes: map[string][]lspTextEdit{
		pathToURI(first):   {{Range: lspRange{Start: lspPosition{Line: 0, Character: 8}, End: lspPosition{Line: 0, Character: 8}}, NewText: " renamed"}},
		pathToURI(missing): {{Range: lspRange{Start: lspPosition{Line: 0}, End: lspPosition{Line: 0, Character: 0}}, NewText: "x"}},
	}})
	if _, err := applyWorkspaceEdit(raw, root); err == nil {
		t.Fatal("expected missing-file preflight error")
	}
	data, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "package main\n" {
		t.Fatalf("first file was partially committed: %q", data)
	}
}

func TestApplyTextEditsRejectsOutOfRangeLine(t *testing.T) {
	_, err := applyTextEdits("one\n", []lspTextEdit{{
		Range: lspRange{Start: lspPosition{Line: 2}, End: lspPosition{Line: 2}},
	}})
	if err == nil || !strings.Contains(err.Error(), "line range") {
		t.Fatalf("expected bounds error, got %v", err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
