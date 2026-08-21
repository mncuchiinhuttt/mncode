package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/skills"
	"os"
	"sort"
	"strings"

	"golang.org/x/term"
)

// OpenInteractiveSkillsBrowser opens a searchable interactive TUI for all skills
func OpenInteractiveSkillsBrowser(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		showSkillsList(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		showSkillsList(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	seenFiles := make(map[string]bool)
	var allSkills []*skills.Skill
	if s.Catalog != nil {
		for _, sk := range s.Catalog.Skills {
			if !seenFiles[sk.FilePath] {
				seenFiles[sk.FilePath] = true
				allSkills = append(allSkills, sk)
			}
		}
	}
	sort.Slice(allSkills, func(i, j int) bool {
		return allSkills[i].Name < allSkills[j].Name
	})

	searchQuery := ""
	selectedIdx := 0
	scrollOffset := 0
	lastLinesCount := 0

	getFiltered := func() []*skills.Skill {
		if strings.TrimSpace(searchQuery) == "" {
			return allSkills
		}
		q := strings.ToLower(searchQuery)
		var res []*skills.Skill
		for _, sk := range allSkills {
			if strings.Contains(strings.ToLower(sk.Name), q) || strings.Contains(strings.ToLower(sk.Description), q) {
				res = append(res, sk)
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

		filtered := getFiltered()
		if selectedIdx >= len(filtered) {
			selectedIdx = len(filtered) - 1
		}
		if selectedIdx < 0 {
			selectedIdx = 0
		}
		maxDisplay := 9
		if selectedIdx < scrollOffset {
			scrollOffset = selectedIdx
		} else if selectedIdx >= scrollOffset+maxDisplay {
			scrollOffset = selectedIdx - maxDisplay + 1
		}

		var lines []string
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("   %s %s",
			BoldCyan("ClaudeKit & Agent Skills Catalog"),
			GrayText(fmt.Sprintf("(%d skills discovered)", len(allSkills)))))
		lines = append(lines, "")

		// Search box
		boxWidth := width - 6
		if boxWidth < 35 {
			boxWidth = 35
		}
		searchContent := searchQuery
		if searchContent == "" {
			searchContent = GrayText("Filter skills (type to search)…")
		}
		pad := boxWidth - 4 - len([]rune(stripAnsi(searchContent)))
		if pad < 0 {
			pad = 0
		}
		lines = append(lines, "  ╭"+strings.Repeat("─", boxWidth)+"╮")
		lines = append(lines, fmt.Sprintf("  │ ⌕ %s%s │", searchContent, strings.Repeat(" ", pad)))
		lines = append(lines, "  ╰"+strings.Repeat("─", boxWidth)+"╯")
		lines = append(lines, "")

		if len(filtered) == 0 {
			lines = append(lines, GrayText("     No matching skills found."))
		} else {
			endIdx := scrollOffset + maxDisplay
			if endIdx > len(filtered) {
				endIdx = len(filtered)
			}

			if scrollOffset > 0 {
				lines = append(lines, GrayText(fmt.Sprintf("    ↑ %d more skills above", scrollOffset)))
			}

			for i := scrollOffset; i < endIdx; i++ {
				sk := filtered[i]
				prefix := "    "
				nameStr := Bold(sk.Name)
				if i == selectedIdx {
					prefix = BoldPastelPink("  ❯ ")
					nameStr = BoldPastelPink(sk.Name)
				}
				desc := truncateText(sk.Description, width-36)
				lines = append(lines, fmt.Sprintf("%s%-26s %s", prefix, nameStr, GrayText(desc)))
			}

			if endIdx < len(filtered) {
				lines = append(lines, GrayText(fmt.Sprintf("    ↓ %d more skills below", len(filtered)-endIdx)))
			}
		}

		lines = append(lines, "")
		lines = append(lines, GrayText("   ↑/↓ to navigate · Enter to activate skill · Esc to exit"))

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

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A': // UP Arrow
				if selectedIdx > 0 {
					selectedIdx--
					render()
				}
				continue
			case 'B': // DOWN Arrow
				filtered := getFiltered()
				if selectedIdx < len(filtered)-1 {
					selectedIdx++
					render()
				}
				continue
			}
		}

		b := buf[0]
		switch b {
		case 3, 27: // Esc or Ctrl+C -> Clean exit
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return

		case 13, 10: // Enter -> Activate selected skill
			filtered := getFiltered()
			if len(filtered) > 0 && selectedIdx < len(filtered) {
				chosen := filtered[selectedIdx]
				if lastLinesCount > 0 {
					fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
				}
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				ActivateSkillByName(s, chosen.Name)
				return
			}
			continue

		case 127, 8: // Backspace
			if len(searchQuery) > 0 {
				searchQuery = searchQuery[:len(searchQuery)-1]
				selectedIdx = 0
				scrollOffset = 0
				render()
			}
			continue

		default:
			if b >= 32 && b <= 126 {
				searchQuery += string(b)
				selectedIdx = 0
				scrollOffset = 0
				render()
			}
		}
	}
}
