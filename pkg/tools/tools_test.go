package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteViewAndEditTools(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	writeTool := &WriteTool{BaseDir: tmpDir}
	viewTool := &ViewTool{BaseDir: tmpDir}
	editTool := &EditTool{BaseDir: tmpDir}

	targetFile := filepath.Join(tmpDir, "test.txt")

	// 1. Test Write
	_, err := writeTool.Execute(ctx, map[string]interface{}{
		"TargetFile":  targetFile,
		"CodeContent": "Hello World\nLine Two\nLine Three",
		"Overwrite":   true,
	})
	if err != nil {
		t.Fatalf("writeTool failed: %v", err)
	}

	// 2. Test View with line range
	viewOut, err := viewTool.Execute(ctx, map[string]interface{}{
		"AbsolutePath": targetFile,
		"StartLine":    float64(2),
		"EndLine":      float64(3),
	})
	if err != nil {
		t.Fatalf("viewTool failed: %v", err)
	}
	if !strings.Contains(viewOut, "2: Line Two") || !strings.Contains(viewOut, "3: Line Three") {
		t.Errorf("unexpected view output: %s", viewOut)
	}

	// 3. Test Edit
	_, err = editTool.Execute(ctx, map[string]interface{}{
		"TargetFile":         targetFile,
		"TargetContent":      "Line Two",
		"ReplacementContent": "Replaced Second Line",
	})
	if err != nil {
		t.Fatalf("editTool failed: %v", err)
	}

	data, _ := os.ReadFile(targetFile)
	if !strings.Contains(string(data), "Replaced Second Line") {
		t.Errorf("expected content to be replaced, got: %s", string(data))
	}
}

func TestGrepAndFindTools(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	file1 := filepath.Join(tmpDir, "sub", "main.go")
	_ = os.MkdirAll(filepath.Dir(file1), 0755)
	_ = os.WriteFile(file1, []byte("package main\nfunc SpecialUniqueFunction() {}\n"), 0644)

	grepTool := &GrepTool{BaseDir: tmpDir}
	grepOut, err := grepTool.Execute(ctx, map[string]interface{}{
		"Query":      "SpecialUniqueFunction",
		"SearchPath": tmpDir,
	})
	if err != nil {
		t.Fatalf("grepTool failed: %v", err)
	}
	if !strings.Contains(grepOut, "SpecialUniqueFunction") {
		t.Errorf("grepTool did not find match: %s", grepOut)
	}

	findTool := &FindTool{BaseDir: tmpDir}
	findOut, err := findTool.Execute(ctx, map[string]interface{}{
		"Pattern":         "main.go",
		"SearchDirectory": tmpDir,
	})
	if err != nil {
		t.Fatalf("findTool failed: %v", err)
	}
	if !strings.Contains(findOut, "main.go") {
		t.Errorf("findTool did not find file: %s", findOut)
	}
}
