package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanCodebaseSkipsSymlinkFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "safe.go"), []byte("package safe\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("safe.go", filepath.Join(root, "linked.go")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	summary, err := ScanCodebase(root)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalFiles != 1 {
		t.Fatalf("scanner counted symlink as source: got %d files", summary.TotalFiles)
	}
}
