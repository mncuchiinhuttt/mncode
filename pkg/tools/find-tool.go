package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindTool searches for files and directories matching a pattern
type FindTool struct {
	BaseDir string
}

func (f *FindTool) Name() string {
	return "find_by_name"
}

func (f *FindTool) Description() string {
	return "Search for files and directories matching a glob pattern or name substring."
}

func (f *FindTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Pattern": map[string]interface{}{
				"type":        "string",
				"description": "Pattern or substring to search for.",
			},
			"SearchDirectory": map[string]interface{}{
				"type":        "string",
				"description": "Directory to search within.",
			},
			"MaxDepth": map[string]interface{}{
				"type":        "integer",
				"description": "Optional maximum depth to search.",
			},
		},
		"required": []string{"Pattern"},
	}
}

func (f *FindTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern, _ := args["Pattern"].(string)
	searchDir, _ := args["SearchDirectory"].(string)
	if searchDir == "" {
		searchDir = "."
	}
	resolvedDir, err := resolveWorkspacePath(f.BaseDir, searchDir, false)
	if err != nil {
		return "", err
	}
	searchDir = resolvedDir

	maxDepth := 10
	if md, ok := args["MaxDepth"].(float64); ok && md > 0 {
		maxDepth = int(md)
	}

	var matches []string
	baseSlashCount := strings.Count(filepath.Clean(searchDir), string(filepath.Separator))

	err = filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || len(matches) >= 50 {
			return nil
		}

		curDepth := strings.Count(filepath.Clean(path), string(filepath.Separator)) - baseSlashCount
		if curDepth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		name := info.Name()
		if info.IsDir() && name != "." && strings.HasPrefix(name, ".") && name != ".claude" {
			return filepath.SkipDir
		}
		if info.IsDir() && (name == "node_modules" || name == "vendor") {
			return filepath.SkipDir
		}

		// Check if matches glob or contains substring
		matched, _ := filepath.Match(pattern, name)
		if matched || strings.Contains(strings.ToLower(name), strings.ToLower(pattern)) {
			rel, relErr := filepath.Rel(searchDir, path)
			if relErr == nil {
				if info.IsDir() {
					matches = append(matches, fmt.Sprintf("[DIR]  %s", rel))
				} else {
					matches = append(matches, fmt.Sprintf("[FILE] %s (%d bytes)", rel, info.Size()))
				}
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed searching in %s: %w", searchDir, err)
	}

	if len(matches) == 0 {
		return fmt.Sprintf("No files found matching '%s' in %s", pattern, searchDir), nil
	}

	return strings.Join(matches, "\n"), nil
}
