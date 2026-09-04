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

func getWorkflowPhases(s *agent.Session, isBrainrot bool) []WorkflowPhase {
	var list []*agent.SubagentRecord
	if s.Subagents != nil {
		list = s.Subagents.List()
	}

	subMap := make(map[string][]string)
	for _, rec := range list {
		lower := strings.ToLower(rec.Name + " " + rec.Role)
		statusStr := rec.Status
		if isBrainrot {
			if statusStr == "running" {
				statusStr = "cooking fr fr [COOK]"
			} else if statusStr == "completed" {
				statusStr = "bussin W [MAX]"
			}
		}
		item := fmt.Sprintf("%s (%s)", rec.Name, statusStr)
		if strings.Contains(lower, "scout") || strings.Contains(lower, "research") {
			subMap["scout"] = append(subMap["scout"], item)
		} else if strings.Contains(lower, "plan") {
			subMap["plan"] = append(subMap["plan"], item)
		} else if strings.Contains(lower, "test") {
			subMap["test"] = append(subMap["test"], item)
		} else if strings.Contains(lower, "review") {
			subMap["review"] = append(subMap["review"], item)
		} else {
			subMap["code"] = append(subMap["code"], item)
		}
	}

	p1Status := "⏳ Pending"
	p2Status := "⏳ Pending"
	if isBrainrot {
		p1Status = "⏳ In queue (delulu)"
		p2Status = "⏳ In queue (delulu)"
	}
	if len(subMap["scout"]) > 0 {
		p1Status = "[ACTIVE] In Progress"
		if isBrainrot {
			p1Status = "[ACTIVE] Rizzing / Cooking fr fr [COOK]"
		}
	}
	if len(subMap["plan"]) > 0 {
		p2Status = "[ACTIVE] In Progress"
		if isBrainrot {
			p2Status = "[ACTIVE] Rizzing / Cooking fr fr [COOK]"
		}
	}

	if isBrainrot {
		return []WorkflowPhase{
			{ID: 1, Name: "1. Snooping Codebase [SCOUT]", Role: "Scouting entrypoints (caught in 4k)", Status: p1Status, Subagents: subMap["scout"], Todos: []string{"Scan entrypoint & AST layout", "Find sus code & tech debt"}, Artifact: "./plans/reports/scout-report.md"},
			{ID: 2, Name: "2. Big Brain Plan [THINK]", Role: "Cooking 200 IQ architecture", Status: p2Status, Subagents: subMap["plan"], Todos: []string{"Write based plan.md (< 80 lines)", "Breakdown grind phases no cap"}, Artifact: "./plans/plan.md"},
			{ID: 3, Name: "3. 10x Sigma Coding [CODE]", Role: "Dropping bussin code (< 200 LOC)", Status: "⏳ In queue", Subagents: subMap["code"], Todos: []string{"Implement real code (no fakes)", "Keep files clean and compilable"}, Artifact: "./docs/project-changelog.md"},
			{ID: 4, Name: "4. Hunting Cringe Bugs [BUG]", Role: "Gaslighting tests into 100% W", Status: "⏳ In queue", Subagents: subMap["test"], Todos: []string{"Run tests with zero cheating", "Verify everything is valid fr"}, Artifact: "./plans/reports/test-results.md"},
			{ID: 5, Name: "5. Final Rizz Audit [SHINE]", Role: "Verifying pure based energy", Status: "⏳ In queue", Subagents: subMap["review"], Todos: []string{"Sign off on ultimate sigma code", "Push updates with maximum aura"}, Artifact: "./docs/development-roadmap.md"},
		}
	}

	return []WorkflowPhase{
		{ID: 1, Name: "1. Scout & Reconnaissance", Role: "Codebase exploration & AST scan", Status: p1Status, Subagents: subMap["scout"], Todos: []string{"Scan entrypoints & project layout", "Map dependencies & symbols"}, Artifact: "./plans/reports/scout-report.md"},
		{ID: 2, Name: "2. Architecture Plan", Role: "Multi-phase implementation design", Status: p2Status, Subagents: subMap["plan"], Todos: []string{"Create plan.md overview (< 80 lines)", "Breakdown phase-01 to phase-N tasks"}, Artifact: "./plans/plan.md"},
		{ID: 3, Name: "3. Code Implementation", Role: "Production code changes (< 200 lines/file)", Status: "⏳ Pending", Subagents: subMap["code"], Todos: []string{"Implement real code following DRY/KISS", "Maintain compile safety"}, Artifact: "./docs/project-changelog.md"},
		{ID: 4, Name: "4. Test & Verification", Role: "Unit, integration & edge-case testing", Status: "⏳ Pending", Subagents: subMap["test"], Todos: []string{"Run test suite without mock cheating", "Verify 100% test pass"}, Artifact: "./plans/reports/test-results.md"},
		{ID: 5, Name: "5. Code Review & Polish", Role: "Quality, security & style audit", Status: "⏳ Pending", Subagents: subMap["review"], Todos: []string{"Check security & performance standards", "Sign-off final implementation"}, Artifact: "./docs/development-roadmap.md"},
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

	isBrainrot := s.Config.GetSetting("brainrot_mode", "false") == "true"
	currentPhaseIdx := 0
	lastLinesCount := 0

	render := func() {
		phases := getWorkflowPhases(s, isBrainrot)
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || width < 70 {
			width = 85
		}
		cardWidth := width - 2
		if cardWidth > 90 {
			cardWidth = 90
		}

		leftColWidth := 34
		rightColWidth := cardWidth - leftColWidth - 5
		if rightColWidth < 20 {
			rightColWidth = 20
		}

		var lines []string
		lines = append(lines, "")

		title := "[ACTION] Ultra Workflow & Subagents Dashboard"
		leftHeader := "Major Workflow Tasks / Phases"
		rightHeader := "Phase Details & Subagents"
		if isBrainrot {
			title = "[ACTION] Ultra Sigma Workflow & Minions Dashboard [THINK]"
			leftHeader = "[TASKS] Sigma Grind Tasks (no cap)"
			rightHeader = "[COOK] Minions Cooking & Rizz"
		}

		dashesCount := cardWidth - DisplayCellWidth(title) - 8
		if dashesCount < 2 {
			dashesCount = 2
		}
		header := fmt.Sprintf("%s %s %s",
			BoldPastelPink("╭── ["), Bold(title),
			BoldPastelPink("] "+strings.Repeat("─", dashesCount)+"╮"))
		lines = append(lines, header)

		splitTop := fmt.Sprintf("│ %s │ %s │",
			PadToCellWidth(BoldCyan(leftHeader), leftColWidth),
			PadToCellWidth(BoldCyan(rightHeader), rightColWidth))
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
				nameStr = TruncateToCellWidth(nameStr, leftColWidth-2)
				leftStr = PadToCellWidth(marker+nameStr, leftColWidth)
			} else {
				leftStr = strings.Repeat(" ", leftColWidth)
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
				subTitle := "Subagents Spawned:"
				if isBrainrot {
					subTitle = "Minions Cooking:"
				}
				rightStr = fmt.Sprintf("%s %s", Bold(subTitle), formatSubagentsCount(selectedPhase.Subagents, isBrainrot))
			case 4:
				if len(selectedPhase.Subagents) > 0 {
					rightStr = fmt.Sprintf("  • %s", strings.Join(selectedPhase.Subagents, ", "))
				} else {
					emptyText := "(None active for this phase yet)"
					if isBrainrot {
						emptyText = "(No minions cooking here yet fr)"
					}
					rightStr = GrayText("  " + emptyText)
				}
			case 5:
				rightStr = fmt.Sprintf("%s %s", GrayText("Artifact:"), GrayText(selectedPhase.Artifact))
			case 6:
				checkTitle := "Phase To-Do Checklist:"
				if isBrainrot {
					checkTitle = "Sigma Grind Checklist:"
				}
				rightStr = Bold(checkTitle)
			case 7:
				if len(selectedPhase.Todos) > 0 {
					rightStr = fmt.Sprintf("  ☑ %s", selectedPhase.Todos[0])
				}
			case 8:
				if len(selectedPhase.Todos) > 1 {
					rightStr = fmt.Sprintf("  ☐ %s", selectedPhase.Todos[1])
				}
			}

			rightPadded := PadToCellWidth(TruncateToCellWidth(rightStr, rightColWidth), rightColWidth)
			lines = append(lines, fmt.Sprintf("│ %s │ %s │", leftStr, rightPadded))
		}

		bottomBorder := fmt.Sprintf("╰%s┴%s╯", strings.Repeat("─", leftColWidth+2), strings.Repeat("─", rightColWidth+2))
		footerText := "[↑/↓] Navigate Phases · [Esc / →] Return to Chat"
		if isBrainrot {
			footerText = "[↑/↓] Navigate Phases · [Esc / →] Back to Chat (Stay sigma)"
		}
		lines = append(lines, bottomBorder, GrayText("  "+footerText))

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

	unregisterCopy := SetCopyCallback(func() { render() })
	defer unregisterCopy()

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

func formatSubagentsCount(list []string, isBrainrot bool) string {
	if len(list) == 0 {
		if isBrainrot {
			return GrayText("0 minions")
		}
		return GrayText("0 active")
	}
	if isBrainrot {
		return BoldGreen(fmt.Sprintf("[MAX] %d cooking", len(list)))
	}
	return BoldGreen(fmt.Sprintf("%d spawned", len(list)))
}
