package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"sort"
	"strings"

	"golang.org/x/term"
)

// OpenSubagentMonitorView opens an interactive Subagents & Workflow Inspector
func OpenSubagentMonitorView(s *agent.Session) {
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

	currentIdx := 0
	showingDetails := false
	lastLinesCount := 0

	render := func() {
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || width < 60 {
			width = 80
		}
		cardWidth := width - 2
		if cardWidth > 85 {
			cardWidth = 85
		}

		var lines []string
		lines = append(lines, "")

		titleLeft := "AUTONOMOUS SUBAGENTS & WORKFLOW MONITOR"
		titleRight := "(→ for chat)"
		midDashes := cardWidth - 8 - len([]rune(titleLeft)) - len([]rune(titleRight))
		if midDashes < 2 {
			midDashes = 2
		}

		topHeader := fmt.Sprintf("%s %s %s %s %s",
			BoldCyan("╭──"), BoldCyan(titleLeft), GrayText(strings.Repeat("─", midDashes)), BoldYellow(titleRight), BoldCyan("──╮"))
		lines = append(lines, topHeader)

		var list []*agent.SubagentRecord
		if s.Subagents != nil {
			list = s.Subagents.List()
		}

		if len(list) == 0 {
			lines = append(lines, Bold("  Available Workspace Subagents:"))
			if s.Catalog != nil && len(s.Catalog.Agents) > 0 {
				var names []string
				for n := range s.Catalog.Agents {
					names = append(names, n)
				}
				sort.Strings(names)

				for _, name := range names {
					ag := s.Catalog.Agents[name]
					roleText := strings.TrimSpace(strings.Split(ag.Role, "\n")[0])
					if roleText == "" || roleText == name {
						roleText = strings.TrimSpace(strings.Split(ag.Description, "\n")[0])
					}
					lines = append(lines, fmt.Sprintf("    • %-22s %s", BoldCyan(name), GrayText(truncateText(roleText, 45))))
				}
			} else {
				lines = append(lines, "    • planner, researcher, scout, tester, debugger, code-reviewer")
			}
			lines = append(lines, "",
				GrayText("  No subagents running in this session yet. Subagents spawn during tasks."),
				GrayText("  Press → (Right Arrow), Esc, or q to return to Chat."))
		} else {
			if currentIdx >= len(list) {
				currentIdx = len(list) - 1
			}
			if currentIdx < 0 {
				currentIdx = 0
			}

			lines = append(lines, fmt.Sprintf("  Active & Completed Agents (%d):", len(list)),
				GrayText("  (Use ↑/↓ to navigate, Enter/Space for logs, → or Esc to return to chat)"), "")

			for i, ag := range list {
				marker := "  "
				if i == currentIdx {
					marker = BoldPastelPink("> ")
				}

				dot := BoldGreen("●")
				tag := BoldGreen("[COMPLETED]")
				if ag.Status == "running" {
					dot = BoldYellow("◯")
					tag = BoldYellow("[RUNNING]")
				} else if ag.Status == "error" {
					dot = BoldRed("✕")
					tag = BoldRed("[ERROR]")
				}

				durStr := ""
				if ag.Duration > 0 {
					durStr = fmt.Sprintf(" %s", GrayText(fmt.Sprintf("(%.1fs)", ag.Duration.Seconds())))
				}
				toolsCount := fmt.Sprintf("%d tools", len(ag.ToolCalls))

				lines = append(lines, fmt.Sprintf("  %s[%d] %s %-16s %-24s %s%s %s",
					marker, i+1, dot, Bold(ag.Name), GrayText("("+ag.Role+")"), tag, durStr, GrayText(toolsCount)))
				lines = append(lines, fmt.Sprintf("        %s %s", GrayText("Task:"), truncateText(ag.Prompt, 60)))

				if i == currentIdx && showingDetails {
					if len(ag.ToolCalls) > 0 {
						lines = append(lines, fmt.Sprintf("        %s %s", BoldCyan("Tools:"), strings.Join(ag.ToolCalls, ", ")))
					}
					if ag.Result != "" {
						lines = append(lines, fmt.Sprintf("        %s %s", BoldGreen("Summary:"), truncateText(ag.Result, 60)))
					}
				}
			}
		}

		lines = append(lines, GrayText("╰"+strings.Repeat("─", cardWidth-2)+"╯"))

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

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A':
				if currentIdx > 0 {
					currentIdx--
					render()
				}
				continue
			case 'B':
				if s.Subagents != nil && currentIdx < len(s.Subagents.List())-1 {
					currentIdx++
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

		b := buf[0]
		switch b {
		case 3, 27, 'q', 'Q', 'c', 'C':
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return

		case 13, 10, ' ':
			showingDetails = !showingDetails
			render()
			continue

		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			idx := int(b - '1')
			if s.Subagents != nil && idx < len(s.Subagents.List()) {
				currentIdx = idx
				showingDetails = true
				render()
			}
			continue
		}
	}
}

func showSubagentsSimple(s *agent.Session) {
	fmt.Println("\nSubagents Monitor:")
	if s.Subagents == nil || len(s.Subagents.List()) == 0 {
		fmt.Println("  No active subagents.")
		return
	}
	for _, ag := range s.Subagents.List() {
		fmt.Printf("  • %s (%s) - %s\n", ag.Name, ag.Role, ag.Status)
	}
	fmt.Println()
}
