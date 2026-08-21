package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditTool replaces specific content in a file
type EditTool struct {
	BaseDir string
}

func (e *EditTool) Name() string {
	return "replace_file_content"
}

func (e *EditTool) Description() string {
	return "Replace a specific contiguous block of text in an existing file."
}

func (e *EditTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"TargetFile": map[string]interface{}{
				"type":        "string",
				"description": "The file path to modify.",
			},
			"TargetContent": map[string]interface{}{
				"type":        "string",
				"description": "The exact string/block to be replaced.",
			},
			"ReplacementContent": map[string]interface{}{
				"type":        "string",
				"description": "The replacement content.",
			},
			"AllowMultiple": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, replace all occurrences.",
			},
		},
		"required": []string{"TargetFile", "TargetContent", "ReplacementContent"},
	}
}

func (e *EditTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path, _ := args["TargetFile"].(string)
	target, _ := args["TargetContent"].(string)
	replacement, _ := args["ReplacementContent"].(string)
	allowMultiple, _ := args["AllowMultiple"].(bool)

	if path == "" || target == "" {
		return "", fmt.Errorf("TargetFile and TargetContent are required")
	}

	if !filepath.IsAbs(path) && e.BaseDir != "" {
		path = filepath.Join(e.BaseDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}

	content := string(data)
	count := strings.Count(content, target)
	if count == 0 {
		return "", fmt.Errorf("TargetContent was not found in %s", path)
	}

	if count > 1 && !allowMultiple {
		return "", fmt.Errorf("TargetContent found %d times in %s; set AllowMultiple to true or provide a more specific chunk", count, path)
	}

	var newContent string
	if allowMultiple {
		newContent = strings.ReplaceAll(content, target, replacement)
	} else {
		newContent = strings.Replace(content, target, replacement, 1)
	}

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write updated file %s: %w", path, err)
	}

	return fmt.Sprintf("Successfully replaced %d occurrence(s) in %s.", count, path), nil
}
