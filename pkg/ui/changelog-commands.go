package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HandleChangelogCommand synthesizes release notes and updates CHANGELOG.md
func HandleChangelogCommand(parts []string, s *agent.Session) {
	logCmd := exec.Command("git", "log", "-n", "35", "--pretty=format:%s (%h)")
	logCmd.Dir = s.WorkspaceDir
	out, err := logCmd.Output()
	if err != nil {
		fmt.Printf("\n%s Git error: %v\n\n", BoldRed("[Error]"), err)
		return
	}

	rawLogs := strings.TrimSpace(string(out))
	if rawLogs == "" {
		fmt.Printf("\n%s No git commits found in repository.\n\n", BoldYellow("[Notice]"))
		return
	}

	var feats, fixes, perfs, docs, chores []string

	for _, line := range strings.Split(rawLogs, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "feat"):
			feats = append(feats, trimmed)
		case strings.HasPrefix(lower, "fix"):
			fixes = append(fixes, trimmed)
		case strings.HasPrefix(lower, "perf") || strings.HasPrefix(lower, "refactor"):
			perfs = append(perfs, trimmed)
		case strings.HasPrefix(lower, "doc"):
			docs = append(docs, trimmed)
		default:
			chores = append(chores, trimmed)
		}
	}

	var sb strings.Builder
	today := time.Now().Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("## Release %s (%s)\n\n", "v0.1.1-beta", today))

	if len(feats) > 0 {
		sb.WriteString("### 🚀 New Features\n")
		for _, f := range feats {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}
	if len(fixes) > 0 {
		sb.WriteString("### 🐛 Bug Fixes\n")
		for _, f := range fixes {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}
	if len(perfs) > 0 {
		sb.WriteString("### ⚡ Performance & Refactoring\n")
		for _, p := range perfs {
			sb.WriteString(fmt.Sprintf("- %s\n", p))
		}
		sb.WriteString("\n")
	}
	if len(docs) > 0 {
		sb.WriteString("### 📝 Documentation\n")
		for _, d := range docs {
			sb.WriteString(fmt.Sprintf("- %s\n", d))
		}
		sb.WriteString("\n")
	}

	changelogMD := sb.String()

	// Append to CHANGELOG.md
	changelogFile := filepath.Join(s.WorkspaceDir, "CHANGELOG.md")
	existing, _ := os.ReadFile(changelogFile)
	newContent := changelogMD + string(existing)
	_ = os.WriteFile(changelogFile, []byte(newContent), 0644)

	fmt.Println()
	fmt.Printf("  %s %s\n", BoldGreen("✓"), Bold("Changelog Synthesized and Appended to CHANGELOG.md!"))
	fmt.Println(GrayText(strings.Repeat("─", 55)))
	fmt.Println(changelogMD)
	_ = CopyToClipboard(changelogMD)
	fmt.Printf("  %s %s\n\n", BoldYellow("Clipboard:"), GrayText("Release notes copied to clipboard!"))
}
