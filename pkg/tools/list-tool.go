package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListTool lists contents of a directory
type ListTool struct {
	BaseDir string
}

func (l *ListTool) Name() string {
	return "list_dir"
}

func (l *ListTool) Description() string {
	return "List files and subdirectories within a given directory."
}

func (l *ListTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"DirectoryPath": map[string]interface{}{
				"type":        "string",
				"description": "Path to list contents of.",
			},
		},
		"required": []string{"DirectoryPath"},
	}
}

func (l *ListTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	dir, _ := args["DirectoryPath"].(string)
	if dir == "" {
		dir = l.BaseDir
	} else if !filepath.IsAbs(dir) && l.BaseDir != "" {
		dir = filepath.Join(l.BaseDir, dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to list directory %s: %w", dir, err)
	}

	var dirs, files []string
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, fmt.Sprintf("[DIR]  %s/", entry.Name()))
		} else {
			files = append(files, fmt.Sprintf("[FILE] %-30s (%d bytes)", entry.Name(), info.Size()))
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Directory: %s (%d items)\n\n", dir, len(entries)))
	for _, d := range dirs {
		sb.WriteString(d + "\n")
	}
	for _, f := range files {
		sb.WriteString(f + "\n")
	}

	return sb.String(), nil
}
