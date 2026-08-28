package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadImageDataRejectsPathOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.png")
	if err := os.WriteFile(outside, []byte("not-an-image"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, ok := loadImageData(workspace, outside); ok {
		t.Fatal("expected image outside workspace to be rejected")
	}
	if _, ok := loadImageData(workspace, "../outside.png"); ok {
		t.Fatal("expected traversal image path to be rejected")
	}
}

func TestLoadImageDataRejectsOversizedFile(t *testing.T) {
	workspace := t.TempDir()
	path := filepath.Join(workspace, "large.png")
	if err := os.WriteFile(path, make([]byte, maxImageAttachmentBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := loadImageData(workspace, "large.png"); ok {
		t.Fatal("expected oversized image to be rejected")
	}
}
