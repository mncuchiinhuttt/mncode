package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"mncode/pkg/artifacts"
)
// ViewTool reads files with line numbering and range slicing
type ViewTool struct {
	BaseDir string
}

func (v *ViewTool) Name() string {
	return "view_file"
}

func (v *ViewTool) Description() string {
	return "View the contents of a file with line numbers. Supports StartLine and EndLine slicing."
}

func (v *ViewTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"AbsolutePath": map[string]interface{}{
				"type":        "string",
				"description": "Path to file to view (absolute or relative to workspace).",
			},
			"StartLine": map[string]interface{}{
				"type":        "integer",
				"description": "Optional start line number (1-indexed, inclusive).",
			},
			"EndLine": map[string]interface{}{
				"type":        "integer",
				"description": "Optional end line number (1-indexed, inclusive).",
			},
		},
		"required": []string{"AbsolutePath"},
	}
}

func (v *ViewTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path, _ := args["AbsolutePath"].(string)
	if path == "" {
		return "", fmt.Errorf("AbsolutePath is required")
	}

	if artifacts.IsVirtualURI(path) {
		return artifacts.ReadVirtualURI(path, v.BaseDir)
	}

	resolvedPath, err := resolveWorkspacePath(v.BaseDir, path, false)
	if err != nil {
		return "", err
	}
	path = resolvedPath

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	var startLine, endLine int
	if s, ok := args["StartLine"].(float64); ok && s > 0 {
		startLine = int(s)
	}
	if e, ok := args["EndLine"].(float64); ok && e > 0 {
		endLine = int(e)
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() {
		line := scanner.Text()
		if (startLine == 0 || lineNum >= startLine) && (endLine == 0 || lineNum <= endLine) {
			lines = append(lines, fmt.Sprintf("%d: %s", lineNum, line))
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file %s: %w", path, err)
	}

	if len(lines) == 0 {
		return fmt.Sprintf("File %s is empty or requested range [%d, %d] contains no lines.", path, startLine, endLine), nil
	}

	return fmt.Sprintf("File Path: %s\nTotal Lines: %d\n\n%s", path, lineNum-1, strings.Join(lines, "\n")), nil
}
