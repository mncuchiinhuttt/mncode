package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveWorkspacePath is the exported wrapper around resolveWorkspacePath
// for callers outside this package (e.g. the desktop app's file preview)
// that need the same workspace-boundary enforcement tool calls get.
func ResolveWorkspacePath(baseDir, rawPath string, allowMissing bool) (string, error) {
	return resolveWorkspacePath(baseDir, rawPath, allowMissing)
}

// resolveWorkspacePath resolves a user-supplied path and keeps it inside the
// configured workspace, including when an existing path traverses a symlink.
// Missing descendants are allowed for tools that create files or directories.
func resolveWorkspacePath(baseDir, rawPath string, allowMissing bool) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.TrimSpace(baseDir) == "" {
		return "", fmt.Errorf("workspace directory is required")
	}

	root, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
		if statErr != nil {
			return "", fmt.Errorf("workspace is unavailable: %w", statErr)
		}
		return "", fmt.Errorf("workspace is not a directory: %s", root)
	}

	candidate := rawPath
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	if resolved, evalErr := filepath.EvalSymlinks(candidate); evalErr == nil {
		if err := ensureWithinWorkspace(root, resolved); err != nil {
			return "", err
		}
		return resolved, nil
	} else if !os.IsNotExist(evalErr) {
		return "", fmt.Errorf("resolve path: %w", evalErr)
	}

	if !allowMissing {
		return "", fmt.Errorf("path does not exist: %s: %w", candidate, os.ErrNotExist)
	}

	return resolveMissingWorkspacePath(root, candidate)
}

func resolveMissingWorkspacePath(root, candidate string) (string, error) {
	missing := make([]string, 0, 4)
	probe := candidate
	for {
		if _, err := os.Lstat(probe); err == nil {
			resolved, evalErr := filepath.EvalSymlinks(probe)
			if evalErr != nil {
				return "", fmt.Errorf("resolve path: %w", evalErr)
			}
			if err := ensureWithinWorkspace(root, resolved); err != nil {
				return "", err
			}
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, missing[index])
			}
			if err := ensureWithinWorkspace(root, resolved); err != nil {
				return "", err
			}
			return resolved, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("resolve path: %w", err)
		}

		parent := filepath.Dir(probe)
		name := filepath.Base(probe)
		if name == "" || parent == probe {
			return "", fmt.Errorf("could not resolve path inside workspace: %s", candidate)
		}
		missing = append(missing, name)
		probe = parent
	}
}

func ensureWithinWorkspace(root, candidate string) error {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return fmt.Errorf("path is outside workspace: %s", candidate)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("path is outside workspace: %s", candidate)
	}
	return nil
}
