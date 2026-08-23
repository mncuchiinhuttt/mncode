package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"

	"golang.org/x/term"
)

// CheckAndPromptStartupUpdate checks for updates and displays an interactive arrow prompt if a new version exists
func CheckAndPromptStartupUpdate(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}

	latestTag, hasUpdate, err := config.CheckLatestVersion()
	if err != nil || !hasUpdate || latestTag == "" {
		return
	}

	options := []string{
		fmt.Sprintf("Yes, update now to %s (recommended)", BoldGreen(latestTag)),
		"Remind me next time",
		"Skip this version",
	}

	selected := 0
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Print("\033[?25l") // hide cursor
	defer fmt.Print("\033[?25h")

	linesRendered := 0
	render := func() {
		if linesRendered > 0 {
			fmt.Printf("\033[%dA\r\033[J", linesRendered)
		}

		var sb strings.Builder
		line1 := fmt.Sprintf(" A new version of mncode is available: %s ➔ %s", GrayText(config.CurrentVersion), BoldGreen(latestTag))
		line2 := " Would you like to update now?"
		sb.WriteString(fmt.Sprintf("%s\r\n", BoldPastelPink("╭── [Update Available] ────────────────────────────────────────────────────────╮")))
		sb.WriteString(fmt.Sprintf("│%s│\r\n", PadToCellWidth(line1, 78)))
		sb.WriteString(fmt.Sprintf("│%s│\r\n", PadToCellWidth(line2, 78)))
		sb.WriteString(fmt.Sprintf("%s\r\n\r\n", BoldPastelPink("╰──────────────────────────────────────────────────────────────────────────────╯")))

		for i, opt := range options {
			if i == selected {
				sb.WriteString(fmt.Sprintf("  %s %s\r\n", BoldCyan("❯"), Bold(opt)))
			} else {
				sb.WriteString(fmt.Sprintf("    %s\r\n", GrayText(opt)))
			}
		}
		sb.WriteString(fmt.Sprintf("\r\n  %s\r\n", GrayText("(Use ↑/↓ arrows to navigate, Enter to confirm, Esc to skip)")))

		out := sb.String()
		linesRendered = strings.Count(out, "\n")
		fmt.Print(out)
	}

	render()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		if buf[0] == 3 { // Ctrl+C
			break
		}
		if buf[0] == 27 && n == 1 { // Esc
			break
		}
		if buf[0] == 13 || buf[0] == 10 { // Enter
			break
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A': // Up
				if selected > 0 {
					selected--
				}
				render()
			case 'B': // Down
				if selected < len(options)-1 {
					selected++
				}
				render()
			}
		} else if n == 1 {
			if buf[0] == 'k' || buf[0] == 'K' {
				if selected > 0 {
					selected--
				}
				render()
			} else if buf[0] == 'j' || buf[0] == 'J' {
				if selected < len(options)-1 {
					selected++
				}
				render()
			} else if buf[0] == 'q' || buf[0] == 'Q' {
				break
			}
		}
	}

	// Clean up prompt
	term.Restore(int(os.Stdin.Fd()), oldState)
	if linesRendered > 0 {
		fmt.Printf("\033[%dA\r\033[J", linesRendered)
	}

	if selected == 0 {
		HandleUpdateCommand(nil, s)
	}
}
