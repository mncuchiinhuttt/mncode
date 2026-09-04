package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWorkspacePathRejectsTraversalAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, rawPath := range []string{
		"../" + filepath.Base(outsideDir) + "/secret.txt",
		outsideFile,
		filepath.Join(root, "..", filepath.Base(outsideDir), "secret.txt"),
	} {
		if _, err := resolveWorkspacePath(root, rawPath, false); err == nil {
			t.Errorf("resolveWorkspacePath(%q) unexpectedly escaped workspace", rawPath)
		}
	}

	link := filepath.Join(root, "linked")
	if err := os.Symlink(outsideDir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := resolveWorkspacePath(root, filepath.Join("linked", "secret.txt"), false); err == nil {
		t.Fatal("resolveWorkspacePath followed a symlink outside the workspace")
	}
}

func TestResolveWorkspacePathPreservesMissingNestedPath(t *testing.T) {
	root := t.TempDir()
	resolved, err := resolveWorkspacePath(root, filepath.Join("new", "nested", "file.txt"), true)
	if err != nil {
		t.Fatalf("resolveWorkspacePath returned error: %v", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(canonicalRoot, "new", "nested", "file.txt")
	if resolved != want {
		t.Fatalf("resolved path = %q, want %q", resolved, want)
	}
}

func TestResolveWorkspacePathAcceptsSymlinkedWorkspaceAlias(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(root, "inside.txt")
	if err := os.WriteFile(inside, []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "workspace-alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	resolved, err := resolveWorkspacePath(alias, filepath.Join(alias, "inside.txt"), false)
	if err != nil {
		t.Fatalf("resolveWorkspacePath rejected a valid workspace alias: %v", err)
	}
	canonicalInside, err := filepath.EvalSymlinks(inside)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != canonicalInside {
		t.Fatalf("resolved path = %q, want %q", resolved, canonicalInside)
	}
}

func TestWriteToolRejectsPathOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "escape.txt")
	tool := &WriteTool{BaseDir: root}

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"TargetFile":  outside,
		"CodeContent": "must not be written",
	})
	if err == nil {
		t.Fatal("WriteTool allowed a path outside the workspace")
	}
	if _, statErr := os.Stat(outside); !os.IsNotExist(statErr) {
		t.Fatalf("outside file exists after rejected write: %v", statErr)
	}
}
func TestViewToolRejectsVirtualURIEscape(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	if err := os.MkdirAll(filepath.Join(root, ".mncode", "scratchpad"), 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := &ViewTool{BaseDir: root}
	_, err := tool.Execute(context.Background(), map[string]interface{}{"AbsolutePath": "local://../outside.txt"})
	if err == nil {
		t.Fatal("ViewTool allowed local URI traversal")
	}
}
