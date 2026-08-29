package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// HandleTreeCommand renders an interactive ASCII file tree of the workspace
func HandleTreeCommand(parts []string, s *agent.Session) {
	maxDepth := 3
	targetDir := s.WorkspaceDir

	if len(parts) > 1 {
		if d, err := strconv.Atoi(parts[1]); err == nil && d > 0 {
			maxDepth = d
		} else {
			targetDir = filepath.Join(s.WorkspaceDir, parts[1])
		}
	}

	fmt.Println()
	rel, _ := filepath.Rel(s.WorkspaceDir, targetDir)
	if rel == "." || rel == "" {
		rel = filepath.Base(s.WorkspaceDir)
	}
	fmt.Printf("%s %s %s\n", BoldPastelPink("[TREE]"), Bold(rel), GrayText(fmt.Sprintf("(depth %d)", maxDepth)))
	fmt.Println(GrayText(strings.Repeat("─", 50)))

	totalFiles, totalDirs := 0, 0
	renderTreeLevel(targetDir, "", 0, maxDepth, &totalFiles, &totalDirs)

	fmt.Println(GrayText(strings.Repeat("─", 50)))
	fmt.Printf("  %s %s\n\n", GrayText("Total:"), Bold(fmt.Sprintf("%d directories, %d files", totalDirs, totalFiles)))
}

func renderTreeLevel(dir string, prefix string, currentDepth, maxDepth int, fileCount, dirCount *int) {
	if currentDepth >= maxDepth {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var visible []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && name != ".env" {
			continue
		}
		if name == "node_modules" || name == "dist" || name == "bin" || name == "build" || name == ".next" {
			continue
		}
		visible = append(visible, e)
	}

	for i, e := range visible {
		isLast := (i == len(visible)-1)
		connector := "├── "
		subPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			subPrefix = prefix + "    "
		}

		fullPath := filepath.Join(dir, e.Name())

		if e.IsDir() {
			*dirCount++
			fmt.Printf("%s%s%s/\n", prefix, connector, BoldCyan(e.Name()))
			renderTreeLevel(fullPath, subPrefix, currentDepth+1, maxDepth, fileCount, dirCount)
		} else {
			*fileCount++
			info, _ := e.Info()
			sizeStr := ""
			if info != nil {
				sizeKB := float64(info.Size()) / 1024.0
				if sizeKB >= 1.0 {
					sizeStr = fmt.Sprintf("%.1fKB", sizeKB)
				} else {
					sizeStr = fmt.Sprintf("%dB", info.Size())
				}
			}

			icon := getFileIcon(e.Name())
			fmt.Printf("%s%s%s %-25s %s\n",
				prefix, connector, icon, Colorize(GetCurrentTheme().Text, e.Name()), GrayText(sizeStr))
		}
	}
}

func getFileIcon(name string) string {
	ext := filepath.Ext(name)
	switch ext {
	case ".go":
		return "🐹"
	case ".ts", ".tsx":
		return "🔷"
	case ".js", ".jsx":
		return "🟨"
	case ".json":
		return "[PLAN]"
	case ".md":
		return "[FILE]"
	case ".py":
		return "🐍"
	case ".rs":
		return "🦀"
	case ".sql":
		return "[DB]"
	case ".svg", ".png", ".jpg":
		return "🖼️"
	case ".sh":
		return "🐚"
	default:
		return "[FILE]"
	}
}
