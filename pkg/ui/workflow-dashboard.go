package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"

	"golang.org/x/term"
)

type WorkflowPhase struct {
	ID        int
	Name      string
	Role      string
	Status    string
	Subagents []string
	Todos     []string
	Artifact  string
}

func getWorkflowPhases(s *agent.Session) []WorkflowPhase {
	var list []*agent.SubagentRecord
	if s.Subagents != nil {
		list = s.Subagents.List()
	}

	subMap := make(map[string][]string)
	for _, rec := range list {
		lower := strings.ToLower(rec.Name + " " + rec.Role)
		if strings.Contains(lower, "scout") || strings.Contains(lower, "research") {
			subMap["scout"] = append(subMap["scout"], fmt.Sprintf("%s (%s)", rec.Name, rec.Status))
		} else if strings.Contains(lower, "plan") {
			subMap["plan"] = append(subMap["plan"], fmt.Sprintf("%s (%s)", rec.Name, rec.Status))
		} else if strings.Contains(lower, "test") {
			subMap["test"] = append(subMap["test"], fmt.Sprintf("%s (%s)", rec.Name, rec.Status))
		} else if strings.Contains(lower, "review") {
			subMap["review"] = append(subMap["review"], fmt.Sprintf("%s (%s)", rec.Name, rec.Status))
		} else {
			subMap["code"] = append(subMap["code"], fmt.Sprintf("%s (%s)", rec.Name, rec.Status))
		}
	}

	p1Status := "⏳ Pending"
	if len(subMap["scout"]) > 0 {
		p1Status = "🟢 In Progress"
	}
	p2Status := "⏳ Pending"
	if len(subMap["plan"]) > 0 {
		p2Status = "🟢 In Progress"
	}

	return []WorkflowPhase{
		{
			ID: 1, Name: "1. Scout & Reconnaissance", Role: "Codebase exploration & AST scan",
			Status: p1Status, Subagents: subMap["scout"],
			Todos:    []string{"Scan entrypoints & project layout", "Map dependencies & symbols"},
			Artifact: "./plans/reports/scout-report.md",
		},
		{
			ID: 2, Name: "2. Architecture Plan", Role: "Multi-phase implementation design",
			Status: p2Status, Subagents: subMap["plan"],
			Todos:    []string{"Create plan.md overview (< 80 lines)", "Breakdown phase-01 to phase-N tasks"},
			Artifact: "./plans/plan.md",
		},
		{
			ID: 3, Name: "3. Code Implementation", Role: "Production code changes (< 200 lines/file)",
			Status: "⏳ Pending", Subagents: subMap["code"],
			Todos:    []string{"Implement real code following DRY/KISS", "Maintain compile safety"},
			Artifact: "./docs/project-changelog.md",
		},
		{
			ID: 4, Name: "4. Test & Verification", Role: "Unit, integration & edge-case testing",
			Status: "⏳ Pending", Subagents: subMap["test"],
			Todos:    []string{"Run test suite without mock cheating", "Verify 100% test pass"},
			Artifact: "./plans/reports/test-results.md",
		},
		{
			ID: 5, Name: "5. Code Review & Polish", Role: "Quality, security & style audit",
			Status: "⏳ Pending", Subagents: subMap["review"],
			Todos:    []string{"Check security & performance standards", "Sign-off final implementation"},
			Artifact: "./docs/development-roadmap.md",
		},
	}
}

// OpenUltraWorkflowMonitorView renders the 2-Pane Split Dashboard for Ultra Workflow & Subagents
func OpenUltraWorkflowMonitorView(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		showSubagentsSimple(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		showSubagentsSimple(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	currentPhaseIdx := 0
	lastLinesCount := 0

	render := func() {
		phases := getWorkflowPhases(s)
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || width < 70 {
			width = 85
		}
		cardWidth := width - 2
		if cardWidth > 90 {
			cardWidth = 90
		}

		leftColWidth := 34
		rightColWidth := cardWidth - leftColWidth - 3

		var lines []string
		lines = append(lines, "")

		title := "⚡ Ultra Workflow & Subagents Dashboard"
		header := fmt.Sprintf("%s %s %s",
			BoldPastelPink("╭── ["), Bold(title),
			BoldPastelPink("] "+strings.Repeat("─", cardWidth-visualLen(title)-10)+"╮"))
		lines = append(lines, header)

		splitTop := fmt.Sprintf("│ %-34s │ %-*s │",
			BoldCyan("Major Workflow Tasks / Phases"), rightColWidth, BoldCyan("Phase Details & Subagents"))
		sep := fmt.Sprintf("├%s┼%s┤", strings.Repeat("─", leftColWidth+2), strings.Repeat("─", rightColWidth+2))
		lines = append(lines, splitTop, sep)

		selectedPhase := phases[currentPhaseIdx]
		maxRows := 10

		for row := 0; row < maxRows; row++ {
			leftStr := ""
			if row < len(phases) {
				p := phases[row]
				marker := "  "
				nameStr := p.Name
				if row == currentPhaseIdx {
					marker = BoldPastelPink("❯ ")
					nameStr = Bold(p.Name)
				}
				if len([]rune(p.Name)) > 30 {
					nameStr = string([]rune(p.Name)[:29]) + "…"
				}
				leftStr = fmt.Sprintf("%s%-32s", marker, nameStr)
			}

			rightStr := ""
			switch row {
			case 0:
				rightStr = fmt.Sprintf("%s %s", Bold("Phase:  "), BoldPastelPink(selectedPhase.Name))
			case 1:
				rightStr = fmt.Sprintf("%s %s", GrayText("Role:   "), GrayText(selectedPhase.Role))
			case 2:
				rightStr = fmt.Sprintf("%s %s", GrayText("Status: "), selectedPhase.Status)
			case 3:
				rightStr = fmt.Sprintf("%s %s", Bold("Subagents Spawned:"), formatSubagentsCount(selectedPhase.Subagents))
			case 4:
				if len(selectedPhase.Subagents) > 0 {
					rightStr = fmt.Sprintf("  • %s", strings.Join(selectedPhase.Subagents, ", "))
				} else {
					rightStr = GrayText("  (None active for this phase yet)")
				}
			case 5:
				rightStr = fmt.Sprintf("%s %s", GrayText("Artifact:"), GrayText(selectedPhase.Artifact))
			case 6:
				rightStr = Bold("Phase To-Do Checklist:")
			case 7:
				if len(selectedPhase.Todos) > 0 {
					rightStr = fmt.Sprintf("  ☑ %s", selectedPhase.Todos[0])
				}
			case 8:
				if len(selectedPhase.Todos) > 1 {
					rightStr = fmt.Sprintf("  ☐ %s", selectedPhase.Todos[1])
				}
			}

			lines = append(lines, fmt.Sprintf("│ %-*s │ %-*s │",
				leftColWidth, leftStr, rightColWidth, truncateText(rightStr, rightColWidth)))
		}

		bottomBorder := fmt.Sprintf("╰%s┴%s╯", strings.Repeat("─", leftColWidth+2), strings.Repeat("─", rightColWidth+2))
		footer := GrayText("  [↑/↓] Navigate Phases · [Esc / →] Return to Chat")
		lines = append(lines, bottomBorder, footer)

		if lastLinesCount > 0 {
			fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
		}

		for i, line := range lines {
			if i < len(lines)-1 {
				fmt.Printf("\r\033[K%s\r\n", line)
			} else {
				fmt.Printf("\r\033[K%s", line)
			}
		}
		lastLinesCount = len(lines)
	}

	render()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		if buf[0] == 3 || (buf[0] == 27 && n == 1) || buf[0] == 'q' || buf[0] == 'Q' {
			break
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A':
				if currentPhaseIdx > 0 {
					currentPhaseIdx--
					render()
				}
				continue
			case 'B':
				if currentPhaseIdx < 4 {
					currentPhaseIdx++
					render()
				}
				continue
			case 'C':
				if lastLinesCount > 0 {
					fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
				}
				return
			}
		}
	}

	if lastLinesCount > 0 {
		fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
	}
}

func formatSubagentsCount(list []string) string {
	if len(list) == 0 {
		return GrayText("0 active")
	}
	return BoldGreen(fmt.Sprintf("%d spawned", len(list)))
}
