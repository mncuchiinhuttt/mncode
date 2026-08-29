package tools

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type workspaceUpdate struct {
	path     string
	original []byte
	updated  []byte
	mode     os.FileMode
	temp     string
}

func prepareWorkspaceUpdate(root, uri string, edits []lspTextEdit) (workspaceUpdate, error) {
	path, err := resolveWorkspacePath(root, uriToPath(uri), false)
	if err != nil {
		return workspaceUpdate{}, fmt.Errorf("rename path rejected: %w", err)
	}
	return prepareWorkspaceUpdatePath(path, edits)
}

func prepareWorkspaceUpdatePath(path string, edits []lspTextEdit) (workspaceUpdate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return workspaceUpdate{}, err
	}
	updated, err := applyTextEdits(string(data), edits)
	if err != nil {
		return workspaceUpdate{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return workspaceUpdate{}, err
	}
	return workspaceUpdate{path: path, original: data, updated: []byte(updated), mode: info.Mode()}, nil
}

func stageWorkspaceUpdate(update *workspaceUpdate) error {
	tmp, err := os.CreateTemp(filepath.Dir(update.path), ".mncode-lsp-rename-*")
	if err != nil {
		return err
	}
	update.temp = tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(update.temp)
		}
	}()
	if err := tmp.Chmod(update.mode.Perm()); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(update.updated); err != nil {
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
	cleanup = false
	return nil
}

func commitWorkspaceUpdates(updates []workspaceUpdate) error {
	defer func() {
		for _, update := range updates {
			if update.temp != "" {
				_ = os.Remove(update.temp)
			}
		}
	}()
	for _, update := range updates {
		current, err := os.ReadFile(update.path)
		if err != nil {
			return fmt.Errorf("recheck rename target %s: %w", update.path, err)
		}
		if !bytes.Equal(current, update.original) {
			return fmt.Errorf("stale rename rejected: %s changed after preflight", update.path)
		}
	}
	for index := range updates {
		if err := replaceExistingFile(updates[index].temp, updates[index].path); err != nil {
			for _, pending := range updates[index:] {
				_ = os.Remove(pending.temp)
			}
			for _, committed := range updates[:index] {
				current, readErr := os.ReadFile(committed.path)
				if readErr == nil && bytes.Equal(current, committed.updated) {
					_ = atomicEditWrite(committed.path, committed.updated, committed.original, committed.mode)
				}
			}
			return fmt.Errorf("commit rename edit %s: %w", updates[index].path, err)
		}
		updates[index].temp = ""
	}
	return nil
}
