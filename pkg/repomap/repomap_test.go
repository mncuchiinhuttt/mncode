package repomap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPageRankConvergence(t *testing.T) {
	// Setup 3 interconnected files: A -> B, B -> C, C -> A
	fileA := &FileNode{Path: "a.go", Symbols: []Symbol{{Name: "FuncA", Kind: KindFunc}}, Refs: []string{"FuncB"}}
	fileB := &FileNode{Path: "b.go", Symbols: []Symbol{{Name: "FuncB", Kind: KindFunc}}, Refs: []string{"FuncC"}}
	fileC := &FileNode{Path: "c.go", Symbols: []Symbol{{Name: "FuncC", Kind: KindFunc}}, Refs: []string{"FuncA"}}

	files := []*FileNode{fileA, fileB, fileC}
	ComputePageRank(files)

	for _, f := range files {
		if f.PageRank <= 0 {
			t.Errorf("expected positive PageRank for %s, got %f", f.Path, f.PageRank)
		}
	}
}

func TestGenerateRepoMap(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-repomap-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create dummy source files
	goCode := `package main
type Server struct {}
func RunServer() {}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	tsCode := `export interface User { id: string; }
export function getUser(): User { return { id: "1" }; }
`
	if err := os.WriteFile(filepath.Join(tempDir, "user.ts"), []byte(tsCode), 0644); err != nil {
		t.Fatal(err)
	}

	repoMap, err := GenerateRepoMap(tempDir, 500)
	if err != nil {
		t.Fatalf("GenerateRepoMap() error = %v", err)
	}

	if !strings.Contains(repoMap, "<repo-map>") || !strings.Contains(repoMap, "</repo-map>") {
		t.Fatalf("missing repo-map tags in output:\n%s", repoMap)
	}
	if !strings.Contains(repoMap, "RunServer") || !strings.Contains(repoMap, "getUser") {
		t.Fatalf("missing symbols in repoMap:\n%s", repoMap)
	}
}
