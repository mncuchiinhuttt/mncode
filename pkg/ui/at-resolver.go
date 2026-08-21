package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveAtContext parses @mentions in user prompt and attaches their real-time content
func ResolveAtContext(workspaceDir string, rawPrompt string) (string, []string) {
	matches := atMentionRegex.FindAllString(rawPrompt, -1)
	if len(matches) == 0 {
		return rawPrompt, nil
	}

	var attachedBadges []string
	var contextBlocks []string
	processedPrompt := rawPrompt
	seen := make(map[string]bool)

	for _, m := range matches {
		if seen[m] {
			continue
		}
		seen[m] = true

		clean := strings.TrimPrefix(m, "@")
		clean = strings.TrimPrefix(clean, "file:")
		clean = strings.TrimPrefix(clean, "folder:")
		clean = strings.TrimPrefix(clean, "dir:")

		switch clean {
		case "git":
			gitCtx, fileCount := getGitContext(workspaceDir)
			if gitCtx != "" {
				contextBlocks = append(contextBlocks, fmt.Sprintf("<git_context>\n%s\n</git_context>", gitCtx))
				attachedBadges = append(attachedBadges, fmt.Sprintf("@git (%d changes)", fileCount))
			}

		case "workspace":
			treeCtx := getWorkspaceTree(workspaceDir)
			if treeCtx != "" {
				contextBlocks = append(contextBlocks, fmt.Sprintf("<workspace_tree>\n%s\n</workspace_tree>", treeCtx))
				attachedBadges = append(attachedBadges, "@workspace")
			}

		default:
			targetPath := filepath.Join(workspaceDir, clean)
			info, err := os.Stat(targetPath)
			if err != nil {
				continue
			}

			if info.IsDir() {
				folderTree := getFolderContents(targetPath, clean)
				contextBlocks = append(contextBlocks, fmt.Sprintf("<folder_context path=\"%s\">\n%s\n</folder_context>", clean, folderTree))
				attachedBadges = append(attachedBadges, fmt.Sprintf("@%s/ (folder)", clean))
			} else {
				content, err := os.ReadFile(targetPath)
				if err == nil {
					lineCount := strings.Count(string(content), "\n") + 1
					contextBlocks = append(contextBlocks, fmt.Sprintf("<file_content path=\"%s\">\n%s\n</file_content>", clean, string(content)))
					attachedBadges = append(attachedBadges, fmt.Sprintf("@%s (%d lines)", clean, lineCount))
				}
			}
		}
	}

	if len(contextBlocks) > 0 {
		processedPrompt = fmt.Sprintf("%s\n\n=== ATTACHED CONTEXT FROM @MENTIONS ===\n%s",
			rawPrompt, strings.Join(contextBlocks, "\n\n"))
	}

	return processedPrompt, attachedBadges
}

func getGitContext(workspaceDir string) (string, int) {
	cmdStatus := exec.Command("git", "status", "-s")
	cmdStatus.Dir = workspaceDir
	statusOut, err := cmdStatus.Output()
	if err != nil {
		return "", 0
	}

	statusLines := strings.TrimSpace(string(statusOut))
	if statusLines == "" {
		return "Working tree clean, no changes.", 0
	}

	lines := strings.Split(statusLines, "\n")
	changeCount := len(lines)

	cmdDiff := exec.Command("git", "diff", "--stat")
	cmdDiff.Dir = workspaceDir
	diffStat, _ := cmdDiff.Output()

	return fmt.Sprintf("Git Status:\n%s\n\nDiff Stat:\n%s", statusLines, string(diffStat)), changeCount
}

func getWorkspaceTree(workspaceDir string) string {
	var entries []string
	_ = filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(workspaceDir, path)
		if rel == "." || strings.HasPrefix(rel, ".") || strings.Contains(rel, "node_modules") || strings.Contains(rel, "dist") {
			if info.IsDir() && rel != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			entries = append(entries, "📁 "+rel+"/")
		} else {
			entries = append(entries, "📄 "+rel)
		}
		if len(entries) >= 60 {
			return filepath.SkipDir
		}
		return nil
	})
	return strings.Join(entries, "\n")
}

func getFolderContents(fullPath string, relPath string) string {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return fmt.Sprintf("Cannot read directory: %v", err)
	}

	var lines []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() {
			lines = append(lines, fmt.Sprintf("📁 %s/%s/", relPath, e.Name()))
		} else {
			lines = append(lines, fmt.Sprintf("📄 %s/%s", relPath, e.Name()))
		}
	}
	return strings.Join(lines, "\n")
}
