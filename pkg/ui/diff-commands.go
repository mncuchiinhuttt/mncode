package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// HandleDiffCommand runs git diff and displays rich uncommitted changes
func HandleDiffCommand(parts []string, s *agent.Session) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		width = 80
	}
	cardWidth := width - 2

	sub := ""
	if len(parts) > 1 {
		sub = strings.ToLower(parts[1])
	}

	// 1. Get git status
	statusCmd := exec.Command("git", "status", "--short")
	statusCmd.Dir = s.WorkspaceDir
	statusOut, err := statusCmd.Output()
	if err != nil {
		fmt.Printf("\n%s Not a git repository or git is not installed in %s\n\n", BoldRed("[Git Error]"), s.WorkspaceDir)
		return
	}

	trimmedStatus := strings.TrimSpace(string(statusOut))
	if trimmedStatus == "" {
		fmt.Println()
		fmt.Printf("  %s %s\n", BoldGreen("[OK]"), Bold("Working tree clean — no uncommitted changes in workspace."))
		fmt.Println()
		return
	}

	// 2. Get git diff
	var diffArgs []string
	diffTitle := "Uncommitted Workspace Diff"
	switch sub {
	case "staged", "cached":
		diffArgs = []string{"diff", "--cached"}
		diffTitle = "Staged Git Changes (--cached)"
	case "head":
		diffArgs = []string{"diff", "HEAD"}
		diffTitle = "All Working + Staged Changes (vs HEAD)"
	default:
		diffArgs = []string{"diff"}
	}

	diffCmd := exec.Command("git", diffArgs...)
	diffCmd.Dir = s.WorkspaceDir
	diffOut, _ := diffCmd.Output()

	// 3. Render Header Banner
	topBorder := fmt.Sprintf("%s %s %s",
		BoldPastelPink("╭── ["),
		BoldCyan(fmt.Sprintf("[SEARCH] %s", diffTitle)),
		BoldPastelPink("] "+strings.Repeat("─", cardWidth-visualLen(diffTitle)-12)+"╮"))

	fmt.Println()
	fmt.Println(topBorder)
	printMCPRow("", cardWidth)

	// 4. File Status Overview
	statusLines := strings.Split(trimmedStatus, "\n")
	printMCPRow(fmt.Sprintf("  %s %s", Bold("Changed Files:"), GrayText(fmt.Sprintf("(%d files modified)", len(statusLines)))), cardWidth)
	for _, sl := range statusLines {
		sl = strings.TrimSpace(sl)
		if sl == "" {
			continue
		}
		prefix := sl[:2]
		filePath := strings.TrimSpace(sl[2:])

		var badge string
		switch {
		case strings.Contains(prefix, "M"):
			badge = BoldYellow("MODIFIED")
		case strings.Contains(prefix, "A"):
			badge = BoldGreen("ADDED   ")
		case strings.Contains(prefix, "D"):
			badge = BoldRed("DELETED ")
		case strings.Contains(prefix, "?"):
			badge = BoldCyan("UNTRACK ")
		default:
			badge = GrayText(prefix)
		}
		printMCPRow(fmt.Sprintf("    [%s] %s", badge, Colorize(GetCurrentTheme().Text, filePath)), cardWidth)
	}

	printMCPRow("", cardWidth)

	// 5. Render Code Diff Chunks if available
	rawDiff := string(diffOut)
	if strings.TrimSpace(rawDiff) != "" {
		t := GetCurrentTheme()
		diffLines := strings.Split(rawDiff, "\n")
		addCount, delCount := 0, 0

		maxShow := 30
		shown := 0

		for _, dl := range diffLines {
			if strings.HasPrefix(dl, "+++") || strings.HasPrefix(dl, "---") || strings.HasPrefix(dl, "diff --git") {
				continue
			}
			if strings.HasPrefix(dl, "+") {
				addCount++
			} else if strings.HasPrefix(dl, "-") {
				delCount++
			}
		}

		printMCPRow(fmt.Sprintf("  %s %s  %s",
			Bold("Diff Content:"),
			BoldGreen(fmt.Sprintf("+%d additions", addCount)),
			BoldRed(fmt.Sprintf("-%d deletions", delCount))), cardWidth)

		for _, dl := range diffLines {
			if strings.HasPrefix(dl, "diff --git") || strings.HasPrefix(dl, "index ") {
				continue
			}
			if shown >= maxShow {
				printMCPRow(fmt.Sprintf("    %s", GrayText(fmt.Sprintf("... [%d more diff lines folded, run 'git diff' for full output]", len(diffLines)-shown))), cardWidth)
				break
			}

			if strings.HasPrefix(dl, "@@") {
				printMCPRow(fmt.Sprintf("    %s", Colorize(t.Info, dl)), cardWidth)
			} else if strings.HasPrefix(dl, "+") {
				printMCPRow(fmt.Sprintf("    %s", Colorize(t.Success, dl)), cardWidth)
			} else if strings.HasPrefix(dl, "-") {
				printMCPRow(fmt.Sprintf("    %s", Colorize(t.Error, dl)), cardWidth)
			} else if len(dl) > 0 {
				printMCPRow(fmt.Sprintf("    %s", GrayText(dl)), cardWidth)
			}
			shown++
		}
	} else {
		printMCPRow(fmt.Sprintf("  %s", GrayText("  (No tracked line diffs. Files may be newly created/untracked)")), cardWidth)
	}

	printMCPRow("", cardWidth)
	fmt.Println(GrayText("╰" + strings.Repeat("─", cardWidth-2) + "╯"))
	fmt.Println()
}
