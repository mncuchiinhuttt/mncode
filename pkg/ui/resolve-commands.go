package ui

import (
	"context"
	"fmt"
	"mncode/pkg/agent"
	"os"
	"path/filepath"
	"strings"
)

// HandleResolveCommand automatically detects and resolves git merge conflicts
func HandleResolveCommand(parts []string, s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldPastelPink("⚔️  Autonomous Git Merge Conflict Resolver"))
	fmt.Println(GrayText(strings.Repeat("─", 65)))
	fmt.Println()

	conflictedFiles := findConflictedFiles(s.WorkspaceDir)
	if len(conflictedFiles) == 0 {
		fmt.Printf("  %s %s\n\n", BoldGreen("✓"), Bold("No git merge conflict markers found in workspace!"))
		return
	}

	fmt.Printf("  %s Found %d conflicted file(s):\n", BoldYellow("!"), len(conflictedFiles))
	for _, f := range conflictedFiles {
		rel, _ := filepath.Rel(s.WorkspaceDir, f)
		fmt.Printf("    • %s\n", BoldRed(rel))
	}
	fmt.Println()

	ctx := context.Background()
	for i, f := range conflictedFiles {
		rel, _ := filepath.Rel(s.WorkspaceDir, f)
		fmt.Printf("  %s Resolving [%d/%d] %s...\n", BoldCyan("🔧"), i+1, len(conflictedFiles), Bold(rel))

		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Printf("    %s Failed to read file: %v\n", BoldRed("✗"), err)
			continue
		}

		resolvePrompt := fmt.Sprintf("Resolve all git merge conflict markers (<<<<<<<, =======, >>>>>>>) in file `%s`.\nHere is the full conflicted file content:\n```\n%s\n```\nAnalyze the intent of both branches, merge the logic cleanly without dropping essential changes, remove all conflict markers, and save the resolved file.", rel, string(content))

		if rErr := s.ProcessUserInput(ctx, resolvePrompt); rErr != nil {
			fmt.Printf("    %s Resolution error: %v\n", BoldRed("✗"), rErr)
		} else {
			// Check if markers still remain
			newContent, _ := os.ReadFile(f)
			if strings.Contains(string(newContent), "<<<<<<<") {
				fmt.Printf("    %s Markers still present in %s. Please inspect manually.\n", BoldYellow("!"), rel)
			} else {
				fmt.Printf("    %s %s resolved successfully!\n", BoldGreen("✓"), Bold(rel))
			}
		}
	}
	fmt.Println()
	fmt.Printf("  %s %s\n\n", BoldGreen("✓ CONFLICT RESOLUTION COMPLETE"), GrayText("Run '/diff' or '/test' to verify changes before committing."))
}

func findConflictedFiles(wsDir string) []string {
	var list []string
	_ = filepath.Walk(wsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.Contains(path, "node_modules") || strings.Contains(path, ".git") || strings.Contains(path, "dist") {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".go" || ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".py" || ext == ".json" || ext == ".md" || ext == ".html" {
			data, err := os.ReadFile(path)
			if err == nil {
				hasStart, hasEq := false, false
				for _, line := range strings.Split(string(data), "\n") {
					trimmed := strings.TrimSpace(line)
					if strings.HasPrefix(trimmed, "<<<<<<< ") || trimmed == "<<<<<<< HEAD" {
						hasStart = true
					}
					if trimmed == "=======" {
						hasEq = true
					}
				}
				if hasStart && hasEq {
					list = append(list, path)
				}
			}
		}
		return nil
	})
	return list
}
