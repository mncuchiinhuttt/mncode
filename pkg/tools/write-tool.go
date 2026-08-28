package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// WriteTool creates or overwrites files
type WriteTool struct {
	BaseDir string
}

func (w *WriteTool) Name() string {
	return "write_to_file"
}

func (w *WriteTool) Description() string {
	return "Create a new file or overwrite an existing file with the provided content."
}

func (w *WriteTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"TargetFile": map[string]interface{}{
				"type":        "string",
				"description": "The file path to write to.",
			},
			"CodeContent": map[string]interface{}{
				"type":        "string",
				"description": "The file content to write.",
			},
			"Overwrite": map[string]interface{}{
				"type":        "boolean",
				"description": "Set to true to overwrite existing files.",
			},
		},
		"required": []string{"TargetFile", "CodeContent"},
	}
}

func (w *WriteTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path, _ := args["TargetFile"].(string)
	content, _ := args["CodeContent"].(string)
	overwrite, _ := args["Overwrite"].(bool)

	if path == "" {
		return "", fmt.Errorf("TargetFile is required")
	}

	resolvedPath, err := resolveWorkspacePath(w.BaseDir, path, true)
	if err != nil {
		return "", err
	}
	path = resolvedPath

	if _, err := os.Stat(path); err == nil && !overwrite {
		return "", fmt.Errorf("file %s already exists; specify Overwrite=true to replace it", path)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s.", len(content), path), nil
}
