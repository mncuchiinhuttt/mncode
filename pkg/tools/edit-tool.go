package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"mncode/pkg/hooks"
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
			"FileHash": map[string]interface{}{
				"type":        "string",
				"description": "Optional sha256 hex of the exact file bytes you read. When supplied and stale, the edit is rejected with the current hash so you can re-read and retry.",
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

// fileFingerprint returns a stable sha256 hex of the raw file bytes.
func fileFingerprint(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// shortHash truncates a hex hash for compact error messages.
func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12] + "…"
	}
	return h
}

func (e *EditTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path, _ := args["TargetFile"].(string)
	if ctx == nil {
		ctx = context.Background()
	}
	target, _ := args["TargetContent"].(string)
	replacement, _ := args["ReplacementContent"].(string)
	allowMultiple, _ := args["AllowMultiple"].(bool)

	if path == "" || target == "" {
		return "", fmt.Errorf("TargetFile and TargetContent are required")
	}

	resolvedPath, err := resolveWorkspacePath(e.BaseDir, path, false)
	if err != nil {
		return "", err
	}
	path = resolvedPath
	release := acquireEditPath(path)
	defer release()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}
	if claimed, _ := args["FileHash"].(string); strings.TrimSpace(claimed) != "" {
		current := fileFingerprint(data)
		if claimed != current {
			return "", fmt.Errorf(
				"stale edit rejected: you supplied FileHash %s but %s is now %s (file changed since you read it); re-read the file and re-apply against fresh content",
				shortHash(claimed), path, shortHash(current),
			)
		}
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

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat updated file %s: %w", path, err)
	}
	if err := atomicEditWrite(path, data, []byte(newContent), info.Mode()); err != nil {
		return "", fmt.Errorf("failed to write updated file %s: %w", path, err)
	}
	hooks.OnPostEdit(ctx, e.BaseDir, path)
	return fmt.Sprintf("Successfully replaced %d occurrence(s) in %s. FileHash: %s", count, path, fileFingerprint([]byte(newContent))), nil
}

// NewHash returns the fingerprint an agent must echo back as FileHash on its
// next edit after a successful write, enabling optimistic-concurrency edits.
func (e *EditTool) NewHash(newContent string) string {
	return fileFingerprint([]byte(newContent))
}
