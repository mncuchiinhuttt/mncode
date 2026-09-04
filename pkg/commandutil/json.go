package commandutil

import (
	"encoding/json"
	"fmt"
	"io"
	"mncode/pkg/artifacts"
	"os"
	"path/filepath"
	"strings"
)

// ReadJSON reads a bounded JSON record into dst.
func ReadJSON(path string, dst any, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = 2 * 1024 * 1024
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > maxBytes {
		return fmt.Errorf("JSON record exceeds %d bytes", maxBytes)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("decode JSON record: %w", err)
	}
	return nil
}

// WritePrivateJSON atomically writes a scrubbed, mode-0600 JSON record.
func WritePrivateJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = []byte(artifacts.ScrubSecrets(string(data)) + "\n")
	dir := filepath.Dir(path)
	if err := ensurePrivateDirectory(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".record-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// WritePrivateJSONExclusive refuses an existing destination before writing.
func WritePrivateJSONExclusive(path string, value any) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("destination already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return WritePrivateJSON(path, value)
}

// Scrub removes known credential formats before display or persistence.
func Scrub(value string) string { return artifacts.ScrubSecrets(value) }
func ensurePrivateDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	current := dir
	for {
		if err := os.Chmod(current, 0o700); err != nil {
			return err
		}
		base := filepath.Base(current)
		if base == ".mncode" {
			return nil
		}
		parent := filepath.Dir(current)
		if filepath.Base(parent) == ".mncode" {
			return os.Chmod(parent, 0o700)
		}
		if !hasPathComponent(current, ".mncode") {
			return nil
		}
		current = parent
	}
}

func hasPathComponent(path, component string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == component {
			return true
		}
	}
	return false
}

// EnsurePrivateDirectory creates a 0700 directory tree for feature artifacts.
func EnsurePrivateDirectory(dir string) error { return ensurePrivateDirectory(dir) }
