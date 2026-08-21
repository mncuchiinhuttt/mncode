package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"

	"golang.org/x/term"
)

// OpenInteractiveUsageDashboard opens the Claude Code styled Usage Dashboard
func OpenInteractiveUsageDashboard(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		ShowSimpleUsage(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		ShowSimpleUsage(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	tabs := []string{"Overview", "Models"}
	activeTab := 0
	timeFilterIdx := 0
	lastLinesCount := 0

	render := func() {
		var lines []string
		lines = append(lines, "")

		// Tab header
		var tabStrs []string
		for i, t := range tabs {
			if i == activeTab {
				tabStrs = append(tabStrs, BoldPastelPink(fmt.Sprintf("[%s]", t)))
			} else {
				tabStrs = append(tabStrs, GrayText(fmt.Sprintf(" %s ", t)))
			}
		}
		lines = append(lines, "  "+strings.Join(tabStrs, "  ")+"  "+GrayText("(Tab/←/→ to switch, 1-3 to filter timeframe, Esc to exit)"))
		lines = append(lines, "")

		if activeTab == 0 {
			lines = append(lines, RenderUsageOverview(s, timeFilterIdx)...)
		} else {
			lines = append(lines, RenderUsageModels(s, timeFilterIdx)...)
		}

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

	buf := make([]byte, 8)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		// Shift+Tab
		if (n == 3 && buf[0] == 27 && buf[1] == 91 && buf[2] == 90) || (n == 2 && buf[0] == 27 && buf[1] == 9) {
			activeTab = (activeTab - 1 + len(tabs)) % len(tabs)
			render()
			continue
		}

		// Arrow keys
		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A': // UP Arrow
				if timeFilterIdx > 0 {
					timeFilterIdx--
				} else {
					timeFilterIdx = 2
				}
				render()
				continue
			case 'B': // DOWN Arrow
				timeFilterIdx = (timeFilterIdx + 1) % 3
				render()
				continue
			case 'C': // RIGHT Arrow -> Next Tab
				activeTab = (activeTab + 1) % len(tabs)
				render()
				continue
			case 'D': // LEFT Arrow -> Prev Tab
				activeTab = (activeTab - 1 + len(tabs)) % len(tabs)
				render()
				continue
			}
		}

		b := buf[0]
		switch b {
		case 3, 27, 'q', 'Q': // Esc, q, Ctrl+C -> Exit without leaving trace
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return

		case 9: // Tab -> Cycle Tab
			activeTab = (activeTab + 1) % len(tabs)
			render()
			continue

		case '1':
			timeFilterIdx = 0
			render()
			continue
		case '2':
			timeFilterIdx = 1
			render()
			continue
		case '3':
			timeFilterIdx = 2
			render()
			continue
		}
	}
}

func ShowSimpleUsage(s *agent.Session) {
	fmt.Println()
	lines := RenderUsageOverview(s, 0)
	for _, l := range lines {
		fmt.Println(l)
	}
	fmt.Println()
}
