package commandutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// Workspace is a canonical, symlink-resolved project root and stable identity.
type Workspace struct {
	Root     string
	Identity string
}

// ResolveWorkspace canonicalizes a workspace and rejects missing/non-directory roots.
func ResolveWorkspace(raw string) (Workspace, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "."
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return Workspace{}, fmt.Errorf("resolve workspace: %w", err)
	}
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return Workspace{}, fmt.Errorf("resolve workspace: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return Workspace{}, fmt.Errorf("stat workspace: %w", err)
	}
	if !info.IsDir() {
		return Workspace{}, fmt.Errorf("workspace is not a directory: %s", root)
	}
	digest := sha256.Sum256([]byte(root))
	return Workspace{Root: root, Identity: hex.EncodeToString(digest[:8])}, nil
}

// Relative returns a normalized workspace-relative path and rejects escapes.
func (w Workspace) Relative(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if resolved, evalErr := filepath.EvalSymlinks(abs); evalErr == nil {
		abs = resolved
	} else if parent, parentErr := filepath.EvalSymlinks(filepath.Dir(abs)); parentErr == nil {
		abs = filepath.Join(parent, filepath.Base(abs))
	}
	rel, err := filepath.Rel(w.Root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}
	return filepath.ToSlash(rel), nil
}

// RejectSymlinkPath rejects symlink components before a destructive operation.
func (w Workspace) RejectSymlinkPath(relative string) error {
	relative = filepath.Clean(filepath.FromSlash(strings.TrimSpace(relative)))
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("path is outside workspace: %s", relative)
	}
	current := w.Root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink path is not allowed: %s", relative)
		}
	}
	return nil
}

var idCounter atomic.Uint64

// NewID returns a filesystem-safe, time-sortable identifier.
func NewID(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "run"
	}
	for _, r := range prefix {
		if r != '-' && r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			prefix = "run"
			break
		}
	}
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), idCounter.Add(1))
}
