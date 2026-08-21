package ui

import (
	"os"
	"path/filepath"
	"strings"
)

// GetGitBranchOrFolder returns the current Git branch name or fallback folder name
func GetGitBranchOrFolder(workspaceDir string) string {
	if workspaceDir == "" {
		workspaceDir = "."
	}
	abs, err := filepath.Abs(workspaceDir)
	if err != nil {
		abs = workspaceDir
	}

	headPath := filepath.Join(abs, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err == nil {
		content := strings.TrimSpace(string(data))
		if strings.HasPrefix(content, "ref: refs/heads/") {
			return strings.TrimPrefix(content, "ref: refs/heads/")
		}
		if len(content) >= 7 {
			return content[:7]
		}
	}

	return filepath.Base(abs)
}
