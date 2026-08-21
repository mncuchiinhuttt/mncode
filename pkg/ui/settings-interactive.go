package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"

	"golang.org/x/term"
)

// OpenInteractiveConfigMenu opens the rich Claude Code Settings & Config Dashboard
func OpenInteractiveConfigMenu(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		showConfigList(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		showConfigList(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	tabs := []string{"Settings", "Status", "Config", "Usage", "Stats"}
	activeTab := 0
	allDefs := GetAllSettingsDefinitions()
	searchQuery := ""
	selectedIdx := 0
	scrollOffset := 0
	lastLinesCount := 0

	getFiltered := func() []SettingDef {
		if strings.TrimSpace(searchQuery) == "" {
			return allDefs
		}
		q := strings.ToLower(searchQuery)
		var res []SettingDef
		for _, d := range allDefs {
			if strings.Contains(strings.ToLower(d.Label), q) || strings.Contains(strings.ToLower(d.Key), q) {
				res = append(res, d)
			}
		}
		return res
	}

	render := func() {
		width := 80
		if w, _, err := term.GetSize(int(os.Stdin.Fd())); err == nil && w > 40 {
			width = w
		}
		if width > 110 {
			width = 110
		}

		var lines []string
		lines = append(lines, "")

		var tabStrs []string
		for i, t := range tabs {
			if i == activeTab {
				tabStrs = append(tabStrs, BoldPastelPink(fmt.Sprintf("[%s]", t)))
			} else {
				tabStrs = append(tabStrs, GrayText(fmt.Sprintf(" %s ", t)))
			}
		}
		lines = append(lines, "  "+strings.Join(tabStrs, "  ")+"  "+GrayText("(Tab to switch tabs, Esc to exit)"))
		lines = append(lines, "")

		if activeTab == 0 {
			var tabLines []string
			tabLines, selectedIdx, scrollOffset = RenderSettingsTab(s, allDefs, searchQuery, selectedIdx, scrollOffset, width)
			lines = append(lines, tabLines...)
		} else if activeTab == 1 {
			lines = append(lines, RenderStatusTab(s)...)
		} else if activeTab == 2 {
			lines = append(lines, RenderConfigTab(s)...)
		} else if activeTab == 3 {
			lines = append(lines, RenderUsageTab(s)...)
		} else if activeTab == 4 {
			lines = append(lines, RenderStatsTab(s)...)
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

		if (n == 3 && buf[0] == 27 && buf[1] == 91 && buf[2] == 90) || (n == 2 && buf[0] == 27 && buf[1] == 9) {
			activeTab = (activeTab - 1 + len(tabs)) % len(tabs)
			render()
			continue
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A': // UP Arrow
				if activeTab == 0 && selectedIdx > 0 {
					selectedIdx--
					render()
				}
				continue
			case 'B': // DOWN Arrow
				filtered := getFiltered()
				if activeTab == 0 && selectedIdx < len(filtered)-1 {
					selectedIdx++
					render()
				}
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
		case 3, 27: // Esc or Ctrl+C -> Exit cleanly without leaving trace
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return

		case 9: // Tab -> Cycle Tab
			activeTab = (activeTab + 1) % len(tabs)
			render()
			continue

		case 13, 10, ' ': // Enter / Space -> Toggle or Cycle Setting
			if activeTab == 0 {
				filtered := getFiltered()
				if len(filtered) > 0 && selectedIdx < len(filtered) {
					chosen := filtered[selectedIdx]
					if chosen.Type == SettingTypeModel {
						if lastLinesCount > 0 {
							fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
						}
						_ = term.Restore(int(os.Stdin.Fd()), oldState)
						OpenInteractiveModelSelector(s)
						oldState, _ = term.MakeRaw(int(os.Stdin.Fd()))
						lastLinesCount = 0
					} else if chosen.Key == "theme" {
						if lastLinesCount > 0 {
							fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
						}
						_ = term.Restore(int(os.Stdin.Fd()), oldState)
						OpenInteractiveThemeSelector(s)
						oldState, _ = term.MakeRaw(int(os.Stdin.Fd()))
						lastLinesCount = 0
					} else {
						ToggleOrCycleSetting(s, chosen)
						_ = config.SaveConfig(s.Config)
					}
					render()
				}
			}
			continue

		case 127, 8: // Backspace
			if activeTab == 0 && len(searchQuery) > 0 {
				searchQuery = searchQuery[:len(searchQuery)-1]
				selectedIdx = 0
				scrollOffset = 0
				render()
			}
			continue

		default:
			if activeTab == 0 && b >= 32 && b <= 126 {
				searchQuery += string(b)
				selectedIdx = 0
				scrollOffset = 0
				render()
			}
		}
	}
}
