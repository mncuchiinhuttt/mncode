package index

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSearchSaveAndStale(t *testing.T) {
	root := t.TempDir()
	writeIndexFile(t, root, "pkg/auth/token.go", "package auth\n\nfunc ValidateToken(input string) bool { return input != \"\" }\n")
	writeIndexFile(t, root, "pkg/ui/view.go", "package ui\n\nfunc RenderHome() string { return \"home\" }\n")
	idx, err := Build(context.Background(), root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	hits := idx.Search(Query{Text: "validate token", Limit: 5})
	if len(hits) == 0 || hits[0].Path != "pkg/auth/token.go" || hits[0].Symbol != "ValidateToken" {
		t.Fatalf("unexpected hits: %+v", hits)
	}
	if err := idx.Save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if hits := loaded.Search(Query{Text: "validate token", Limit: 5}); len(hits) == 0 || hits[0].Path != "pkg/auth/token.go" {
		t.Fatalf("reloaded index lost postings: %+v", hits)
	}
	writeIndexFile(t, root, "pkg/new.go", "package new\nfunc Added() {}\n")
	if _, err := Open(root); !errors.Is(err, ErrStale) {
		t.Fatalf("expected new-file stale error, got %v", err)
	}
	writeIndexFile(t, root, "pkg/auth/token.go", "package auth\n\nfunc ValidateToken(input string) bool { return true }\n")
	if _, err := Open(root); !errors.Is(err, ErrStale) {
		t.Fatalf("expected changed-file stale error, got %v", err)
	}
}

func TestSecretAndBuildDirectoriesExcluded(t *testing.T) {
	root := t.TempDir()
	writeIndexFile(t, root, ".env.go", "package secret\nfunc APIKey() {}\n")
	writeIndexFile(t, root, "node_modules/pkg.js", "export function Hidden() {}\n")
	idx, err := Build(context.Background(), root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Documents) != 0 {
		t.Fatalf("excluded files indexed: %+v", idx.Documents)
	}
}

func writeIndexFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
