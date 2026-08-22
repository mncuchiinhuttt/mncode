package ui

import (
	"bufio"
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"

	"golang.org/x/term"
)

// OpenInteractiveModelSelector opens a dynamic interactive model browser based on available accounts & free tiers
func OpenInteractiveModelSelector(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		showCurrentModel(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		showCurrentModel(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	modelsList := GetAvailableModels(s)
	numItems := len(modelsList)

	currentIdx := 0
	for i, m := range modelsList {
		if strings.EqualFold(m.ID, s.Config.Model) {
			currentIdx = i
			break
		}
	}

	lastLinesCount := 0

	render := func() {
		var lines []string
		lines = append(lines, BoldCyan("mncode Model Selector:"))
		lines = append(lines, GrayText("  (Use Up/Down or 1-9 to navigate, Enter to apply, Esc/q to cancel)"))
		lines = append(lines, "")

		for i, m := range modelsList {
			marker := "  "
			nameColor := m.Name
			if i == currentIdx {
				marker = BoldGreen("> ")
				nameColor = Bold(m.Name)
			}
			activeBadge := ""
			if strings.EqualFold(m.ID, s.Config.Model) {
				activeBadge = BoldGreen(" [CURRENT]")
			}
			tagStr := BoldYellow(m.Tag)
			if m.Tag == "[Antigravity]" {
				tagStr = BoldGreen(m.Tag)
			} else if m.Tag == "[OpenCode Free]" || m.Tag == "[OpenRouter Free]" {
				tagStr = BoldCyan(m.Tag)
			}

			lines = append(lines, fmt.Sprintf("  %s[%2d] %-30s %-18s%s",
				marker, i+1, nameColor, tagStr, activeBadge))
		}

		selected := modelsList[currentIdx]
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s %s", BoldCyan("Details:"), selected.Description))
		if selected.Provider != "" {
			lines = append(lines, fmt.Sprintf("  %s %s | %s %s",
				GrayText("Model ID:"), Bold(selected.ID),
				GrayText("Provider:"), BoldCyan(string(selected.Provider))))
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

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A': // Up
				if currentIdx > 0 {
					currentIdx--
				} else {
					currentIdx = numItems - 1
				}
				render()
				continue
			case 'B': // Down
				if currentIdx < numItems-1 {
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
		case 3, 27, 'q', 'Q':
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return

		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			idx := int(b - '1')
			if idx < numItems {
				currentIdx = idx
				render()
			}
			continue

		case 13, 10, ' ':
			chosen := modelsList[currentIdx]
			if chosen.ID == "custom" {
				fmt.Print("\r\n\033[J\r\n")
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				promptCustomModel(s)
				return
			}

			s.Config.Model = chosen.ID
			if chosen.Provider != "" {
				s.Config.Provider = chosen.Provider
			}
			_ = config.SaveConfig(s.Config)
			_ = s.EnsureProvider()

			fmt.Print("\r\n\033[J\r\n")
			fmt.Printf("%s Switched model to %s (Provider: %s)\r\n\r\n",
				BoldGreen("[Success]"),
				Bold(chosen.Name),
				BoldCyan(string(s.Config.Provider)))
			return
		}
	}
}

func promptCustomModel(s *agent.Session) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter custom model name (e.g. gemini-3.7-flash-high, mistral-large): ")
	modelInput, _ := reader.ReadString('\n')
	modelInput = strings.TrimSpace(modelInput)
	if modelInput == "" {
		fmt.Println("Cancelled.")
		return
	}
	s.Config.Model = modelInput
	_ = config.SaveConfig(s.Config)
	_ = s.EnsureProvider()
	fmt.Printf("\n%s Switched model to %s\n\n", BoldGreen("[Success]"), Bold(modelInput))
}

func showCurrentModel(s *agent.Session) {
	fmt.Printf("\nCurrent Model: %s (Provider: %s)\n", BoldCyan(s.Config.Model), s.Config.Provider)
	fmt.Println(GrayText("Usage: /model <name> to change directly, or run /model in an interactive terminal.\n"))
}
