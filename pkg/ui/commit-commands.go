package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os/exec"
	"path/filepath"
	"strings"
)

// HandleCommitCommand performs AI-assisted semantic conventional commits
func HandleCommitCommand(parts []string, s *agent.Session) {
	// 1. Check git status
	statusCmd := exec.Command("git", "status", "--short")
	statusCmd.Dir = s.WorkspaceDir
	out, err := statusCmd.Output()
	if err != nil {
		fmt.Printf("\n%s Not a git repository: %v\n\n", BoldRed("[Git Error]"), err)
		return
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		fmt.Printf("\n%s Nothing to commit, working tree clean.\n\n", BoldGreen("✓"))
		return
	}

	// 2. Determine Commit Message
	var commitMsg string
	shouldPush := false

	for i := 1; i < len(parts); i++ {
		p := parts[i]
		if p == "-m" && i+1 < len(parts) {
			commitMsg = strings.Join(parts[i+1:], " ")
			break
		} else if p == "--push" || p == "-p" || p == "push" {
			shouldPush = true
		}
	}

	if commitMsg == "" {
		commitMsg = generateSemanticCommitMessage(trimmed, s.WorkspaceDir)
	}

	// 3. Stage changes
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = s.WorkspaceDir
	if addErr := addCmd.Run(); addErr != nil {
		fmt.Printf("\n%s Failed to stage changes: %v\n\n", BoldRed("[Error]"), addErr)
		return
	}

	// 4. Commit
	cCmd := exec.Command("git", "commit", "-m", commitMsg)
	cCmd.Dir = s.WorkspaceDir
	cOut, cErr := cCmd.CombinedOutput()
	if cErr != nil {
		fmt.Printf("\n%s Git commit failed: %s\n\n", BoldRed("[Error]"), string(cOut))
		return
	}

	fmt.Println()
	fmt.Printf("  %s %s\n", BoldGreen("✓"), Bold("Semantic Commit Created:"))
	fmt.Printf("    %s %s\n", BoldCyan("Message:"), Colorize(GetCurrentTheme().Primary, commitMsg))

	if shouldPush {
		fmt.Printf("  %s Pushing to remote...\n", BoldCyan("🚀"))
		pushCmd := exec.Command("git", "push")
		pushCmd.Dir = s.WorkspaceDir
		if pOut, pErr := pushCmd.CombinedOutput(); pErr != nil {
			fmt.Printf("  %s Push failed: %s\n", BoldRed("✗"), string(pOut))
		} else {
			fmt.Printf("  %s %s\n", BoldGreen("✓"), Bold("Successfully pushed to remote!"))
		}
	}
	fmt.Println()
}

func generateSemanticCommitMessage(gitStatus, workspaceDir string) string {
	lines := strings.Split(gitStatus, "\n")
	var touchedFiles []string
	hasDocs, hasTests, hasFeat, hasFix := false, false, false, false

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if len(trimmed) < 3 {
			continue
		}
		fPath := strings.TrimSpace(trimmed[2:])
		base := filepath.Base(fPath)
		touchedFiles = append(touchedFiles, base)

		lower := strings.ToLower(fPath)
		if strings.Contains(lower, "doc") || strings.HasSuffix(lower, ".md") {
			hasDocs = true
		} else if strings.Contains(lower, "test") || strings.HasSuffix(lower, "_test.go") {
			hasTests = true
		} else if strings.Contains(lower, "fix") || strings.Contains(lower, "bug") {
			hasFix = true
		} else {
			hasFeat = true
		}
	}

	prefix := "feat"
	if hasFix && !hasFeat {
		prefix = "fix"
	} else if hasTests && !hasFeat {
		prefix = "test"
	} else if hasDocs && !hasFeat {
		prefix = "docs"
	}

	scope := "core"
	if len(touchedFiles) == 1 {
		scope = strings.TrimSuffix(touchedFiles[0], filepath.Ext(touchedFiles[0]))
	} else if len(touchedFiles) > 1 && len(touchedFiles) <= 3 {
		scope = strings.Join(touchedFiles, ", ")
	}

	return fmt.Sprintf("%s(%s): update and enhance functionality across %d file(s)", prefix, scope, len(touchedFiles))
}
