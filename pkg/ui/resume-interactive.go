package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"

	"golang.org/x/term"
)

// OpenInteractiveResumePicker opens an interactive TUI to browse and resume saved sessions
func OpenInteractiveResumePicker(s *agent.Session) {
	sessions, err := agent.ListSavedSessions()
	if err != nil || len(sessions) == 0 {
		fmt.Printf("\n%s No previous chat sessions found to resume.\n\n", BoldYellow("[Notice]"))
		return
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		latest, _ := agent.GetLatestSavedSession()
		if latest != nil {
			s.Restore(latest)
			fmt.Printf("\n%s Resumed latest session: %s (%d turns)\n\n", BoldGreen("[Resumed]"), latest.Title, latest.Turns)
		}
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	currentIdx := 0
	lastLinesCount := 0
	totalItems := len(sessions)
	if totalItems > 10 {
		totalItems = 10
	}

	for {
		var lines []string
		lines = append(lines, "", BoldCyan("   Resume Previous Chat Sessions:"),
			GrayText("   (Use Up/Down or 1-9 to navigate, Enter to resume, Esc to cancel)"), "")

		for i := 0; i < totalItems; i++ {
			sess := sessions[i]
			prefix := "    "
			titleStr := sess.Title
			if i == currentIdx {
				prefix = BoldPastelPink("  ❯ ")
				titleStr = Bold(sess.Title)
			}
			timeStr := sess.UpdatedAt.Format("01/02 15:04")
			turnsStr := fmt.Sprintf("(%d turns)", sess.Turns)
			modelStr := sess.Model
			if modelStr == "" {
				modelStr = "default"
			}

			lines = append(lines, fmt.Sprintf("%s[%d] \033[90m%s\033[0m %-36s \033[36m%-10s\033[0m \033[90m· %s\033[0m",
				prefix, i+1, timeStr, truncateText(titleStr, 36), turnsStr, modelStr))
		}

		lines = append(lines, "", fmt.Sprintf("   %s %d saved sessions found", GrayText("Total:"), len(sessions)),
			GrayText("   Enter to restore conversation · Esc to cancel"))

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

		buf := make([]byte, 3)
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A':
				if currentIdx > 0 {
					currentIdx--
				} else {
					currentIdx = totalItems - 1
				}
				continue
			case 'B':
				if currentIdx < totalItems-1 {
					currentIdx++
				} else {
					currentIdx = 0
				}
				continue
			}
		}

		b := buf[0]
		switch b {
		case 3, 27, 'q', 'Q':
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return

		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			choice := int(b - '1')
			if choice < totalItems {
				currentIdx = choice
			}
			continue

		case 13, 10, ' ':
			chosen := sessions[currentIdx]
			s.Restore(chosen)
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			fmt.Println()
			fmt.Printf("%s Session restored: %s\r\n", BoldGreen("[Resumed]"), Bold(chosen.Title))
			fmt.Printf("  \033[90mLoaded %d turns (%d messages) · Model: %s\033[0m\r\n",
				chosen.Turns, len(chosen.Messages), BoldCyan(chosen.Model))
			RenderResumedHistory(chosen.Messages)
			return
		}
	}
}

// HandleResumeCommand handles /resume slash command
func HandleResumeCommand(parts []string, s *agent.Session) {
	if len(parts) > 1 {
		arg := strings.TrimSpace(parts[1])
		if arg == "--last" || arg == "last" || arg == "-l" {
			latest, err := agent.GetLatestSavedSession()
			if err != nil {
				fmt.Printf("%s %v\n", BoldRed("[Error]"), err)
				return
			}
			s.Restore(latest)
			fmt.Printf("\n%s Resumed latest session: %s (%d turns)\n", BoldGreen("[Resumed]"), Bold(latest.Title), latest.Turns)
			RenderResumedHistory(latest.Messages)
			return
		}
		// Load by session ID
		saved, err := agent.LoadSavedSession(arg)
		if err != nil {
			fmt.Printf("%s Could not find session '%s': %v\n", BoldRed("[Error]"), arg, err)
			return
		}
		s.Restore(saved)
		fmt.Printf("\n%s Resumed session: %s (%d turns)\n", BoldGreen("[Resumed]"), Bold(saved.Title), saved.Turns)
		RenderResumedHistory(saved.Messages)
		return
	}
	OpenInteractiveResumePicker(s)
}
