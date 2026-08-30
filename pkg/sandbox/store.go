package sandbox

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"mncode/pkg/commandutil"
)

// View loads a persisted run manifest.
func (h *Harness) View(ctx context.Context, runID string) (RunResult, error) {
	if err := contextErr(ctx); err != nil {
		return RunResult{}, err
	}
	if _, err := safeID(runID); err != nil {
		return RunResult{}, err
	}
	if err := h.Workspace.RejectSymlinkPath(filepath.Join(h.RunDir, runID, "manifest.json")); err != nil {
		return RunResult{}, err
	}
	path, err := h.path(filepath.Join(h.RunDir, runID, "manifest.json"))
	if err != nil {
		return RunResult{}, err
	}
	var result RunResult
	if err := commandutil.ReadJSON(path, &result, 2*1024*1024); err != nil {
		return RunResult{}, err
	}
	if result.ID != runID || result.SchemaVersion != 1 {
		return RunResult{}, fmt.Errorf("malformed sandbox run %q", runID)
	}
	return result, nil
}

// Clean removes one run only after explicit approval.
func (h *Harness) Clean(ctx context.Context, runID string, approved bool) error {
	if !approved {
		return fmt.Errorf("cleaning sandbox run requires explicit approval")
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	if _, err := safeID(runID); err != nil {
		return err
	}
	relative := filepath.Join(h.RunDir, runID)
	if err := h.Workspace.RejectSymlinkPath(relative); err != nil {
		return err
	}
	path, err := h.path(relative)
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("sandbox run path is not a directory")
	}
	return os.RemoveAll(path)
}

func copyFixture(source, destination string, limits commandutil.Limits) error {
	count := 0
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		if entry.Name() == "fixture.json" && path == filepath.Join(source, entry.Name()) {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("fixture contains symlink %q", entry.Name())
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if secretFixturePath(rel) {
			return fmt.Errorf("fixture contains secret-like file %q", rel)
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("fixture contains non-regular file %q", rel)
		}
		count++
		if count > limits.MaxFiles {
			return fmt.Errorf("fixture exceeds %d files", limits.MaxFiles)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > limits.MaxFileBytes {
			return fmt.Errorf("fixture file %q exceeds size limit", rel)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, io.LimitReader(in, limits.MaxFileBytes+1))
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		return os.Chmod(target, 0o600)
	})
}

// ValidateRunID accepts only private path components.
func ValidateRunID(id string) error {
	id = strings.TrimSpace(id)
	_, err := safeID(id)
	return err
}
func secretFixturePath(path string) bool {
	lower := strings.ToLower(filepath.Base(path))
	return strings.HasPrefix(lower, ".env") || strings.Contains(lower, "credential") ||
		strings.Contains(lower, "secret") || strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key")
}
