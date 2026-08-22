package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"

	"golang.org/x/term"
)

// OpenInteractiveThemeSelector opens a 2-column theme picker with live code diff preview & full-line bg toggle
func OpenInteractiveThemeSelector(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		showThemeList()
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		showThemeList()
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	currentThemeID := s.Config.GetSetting("theme", "pastel-pink")
	currentIdx := 0
	for i, key := range ThemeList {
		if strings.EqualFold(key, currentThemeID) || strings.EqualFold(Themes[key].Name, currentThemeID) {
			currentIdx = i
			break
		}
	}

	fullLineBg := s.Config.GetSetting("diff_style", "Full-line Background") != "Syntax Text Only"
	lastLinesCount := 0

	render := func() {
		activeTheme := Themes[ThemeList[currentIdx]]

		var lines []string
		lines = append(lines, "")
		lines = append(lines, BoldCyan("   Choose a Theme")+" "+GrayText("· (Use Up/Down or 1-7 to preview, Tab/b to toggle diff style, Enter to apply)"))
		lines = append(lines, "")

		// Build Left Column (Fixed 38 visual columns for 7 themes)
		numThemes := len(ThemeList)
		leftCol := make([]string, 9)
		leftWidth := 38
		for i := 0; i < numThemes; i++ {
			thm := Themes[ThemeList[i]]
			prefix := "    "
			nameStr := thm.Name
			if i == currentIdx {
				prefix = BoldPastelPink("  ❯ ")
				nameStr = Bold(thm.Name)
			}
			activeBadge := ""
			if strings.EqualFold(ThemeList[i], currentThemeID) || strings.EqualFold(thm.Name, currentThemeID) {
				activeBadge = " " + BoldGreen("[active]")
			}
			rawLeft := fmt.Sprintf("%s[%d] %s%s", prefix, i+1, nameStr, activeBadge)
			cleanLeft := stripAnsi(rawLeft)
			pad := leftWidth - len([]rune(cleanLeft))
			if pad < 0 {
				pad = 0
			}
			leftCol[i] = rawLeft + strings.Repeat(" ", pad)
		}
		leftCol[7] = strings.Repeat(" ", leftWidth)
		rawPalette := fmt.Sprintf("    Palette: %s", RenderThemeSwatch(activeTheme))
		padPal := leftWidth - len([]rune(stripAnsi(rawPalette)))
		if padPal < 0 {
			padPal = 0
		}
		leftCol[8] = rawPalette + strings.Repeat(" ", padPal)

		// Build Right Column (Live Code Diff Preview with 9 rows)
		t := activeTheme
		rightCol := make([]string, 9)
		rightWidth := 52
		rightCol[0] = fmt.Sprintf("%s%s%s", Colorize(t.Muted, "╭── "), Colorize(AttrBold+t.Primary, "agent_workflow.go"), Colorize(t.Muted, " ──────────── (live diff preview)"))
		rightCol[1] = fmt.Sprintf("%s  %s %s", Colorize(t.Muted, "│"), Colorize(t.Info, "func"), Colorize(t.Text, "ExecuteTask(ctx context.Context) error {"))

		delText := "-    mode := \"direct\""
		addText := "+    mode := \"ultra-workflow\""
		if fullLineBg && t.DiffDelBg != "" {
			padDel := rightWidth - len([]rune(delText)) - 4
			if padDel < 0 {
				padDel = 0
			}
			rightCol[2] = fmt.Sprintf("%s %s", Colorize(t.Muted, "│"), Colorize(t.DiffDelBg+t.DiffDelFg, fmt.Sprintf(" %s%s ", delText, strings.Repeat(" ", padDel))))
		} else {
			rightCol[2] = fmt.Sprintf("%s  %s", Colorize(t.Muted, "│"), Colorize(t.Error, delText))
		}

		if fullLineBg && t.DiffAddBg != "" {
			padAdd := rightWidth - len([]rune(addText)) - 4
			if padAdd < 0 {
				padAdd = 0
			}
			rightCol[3] = fmt.Sprintf("%s %s", Colorize(t.Muted, "│"), Colorize(t.DiffAddBg+t.DiffAddFg, fmt.Sprintf(" %s%s ", addText, strings.Repeat(" ", padAdd))))
		} else {
			rightCol[3] = fmt.Sprintf("%s  %s", Colorize(t.Muted, "│"), Colorize(t.Success, addText))
		}

		rightCol[4] = fmt.Sprintf("%s  %s", Colorize(t.Muted, "│"), Colorize(t.Muted, "     return s.RunAutonomousLoop(mode)"))
		rightCol[5] = fmt.Sprintf("%s  %s", Colorize(t.Muted, "│"), Colorize(t.Text, "}"))
		rightCol[6] = fmt.Sprintf("%s  %s", Colorize(t.Muted, "│"), Colorize(t.Muted, "// Theme: "+t.Name))
		rightCol[7] = Colorize(t.Muted, "╰─────────────────────────────────────────────────────────")
		rightCol[8] = fmt.Sprintf("  %s %s", Colorize(AttrBold+t.Success, "[OK]"), Colorize(t.Muted, "replace_file_content (1 line modified)"))

		// Merge 2 columns with straight vertical divider
		for r := 0; r < 9; r++ {
			div := Colorize(activeTheme.Muted, "│ ")
			lines = append(lines, fmt.Sprintf("%s%s%s", leftCol[r], div, rightCol[r]))
		}

		diffStyleName := "Full-line Background"
		if !fullLineBg {
			diffStyleName = "Syntax Text Only"
		}

		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("   %s %-30s %s %s",
			Bold("Theme:      "), Colorize(AttrBold+activeTheme.Primary, activeTheme.Name),
			Bold("Diff Style:"), BoldCyan(diffStyleName)))
		lines = append(lines, fmt.Sprintf("   %s %s", GrayText("Description:"), activeTheme.Description))
		lines = append(lines, "")
		lines = append(lines, GrayText("   ↑/↓ to navigate · Tab/b to toggle diff style · Enter to confirm · Esc to exit"))

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
			case 'A': // UP Arrow
				if currentIdx > 0 {
					currentIdx--
				} else {
					currentIdx = len(ThemeList) - 1
				}
				render()
				continue
			case 'B': // DOWN Arrow
				if currentIdx < len(ThemeList)-1 {
					currentIdx++
				} else {
					currentIdx = 0
				}
				render()
				continue
			}
		}

		b := buf[0]
		switch b {
		case 3, 27, 'q', 'Q': // Esc, q -> Cancel without trace
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return

		case 9, 'b', 'B': // Tab / 'b' -> Toggle diff highlight mode
			fullLineBg = !fullLineBg
			if fullLineBg {
				s.Config.SetSetting("diff_style", "Full-line Background")
			} else {
				s.Config.SetSetting("diff_style", "Syntax Text Only")
			}
			_ = config.SaveConfig(s.Config)
			render()
			continue

		case 13, 10: // Enter -> Apply theme
			chosenKey := ThemeList[currentIdx]
			chosen := Themes[chosenKey]
			SetTheme(chosenKey)
			s.Config.SetSetting("theme", chosenKey)
			_ = config.SaveConfig(s.Config)

			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			fmt.Printf("%s Switched theme to: %s %s\r\n\r\n",
				BoldGreen("[Success]"),
				Bold(chosen.Name),
				RenderThemeSwatch(chosen))
			return

		case '1', '2', '3', '4', '5', '6', '7':
			idx := int(b - '1')
			if idx >= 0 && idx < len(ThemeList) {
				currentIdx = idx
				render()
			}
			continue
		}
	}
}

func showThemeList() {
	fmt.Println("\nAvailable Themes:")
	for i, key := range ThemeList {
		thm := Themes[key]
		fmt.Printf("  [%d] %-24s %s - %s\n", i+1, thm.Name, RenderThemeSwatch(thm), thm.Description)
	}
	fmt.Println()
}
