package ui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type DebateModelItem struct {
	ID   string
	Name string
}

var defaultDebateModelItems = []DebateModelItem{
	{"claude-sonnet-4-6", "Claude Sonnet 4.6 (Thinking)"},
	{"deepseek-reasoner", "DeepSeek R1 (Reasoner)"},
	{"gemini-3.7-flash-high", "Gemini 3.7 Flash (High Thinking)"},
	{"o3", "OpenAI o3 (Reasoning Flagship)"},
	{"claude-opus-4-6-thinking", "Claude Opus 4.6 (Deep Reasoning)"},
	{"gemini-pro-agent", "Gemini 3.1 Pro (Deep Agent Tier)"},
	{"gpt-4.5", "GPT-4.5 Orion"},
	{"gpt-4o", "GPT-4o Omni"},
}

// selectDebateModelsInteractive runs an interactive multi-select where Space toggles the green dot.
func selectDebateModelsInteractive() []string {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return []string{"claude-sonnet-4-6", "deepseek-reasoner"}
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return []string{"claude-sonnet-4-6", "deepseek-reasoner"}
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	cursor := 0
	selected := make(map[int]bool)
	// Default pre-select first two models
	selected[0] = true
	selected[1] = true

	lastLinesCount := 0

	render := func() {
		var lines []string
		lines = append(lines, "\033[1;36m[STEP 2: SELECT DEBATER MODELS]\033[0m")
		lines = append(lines, "\033[2m  [Space/1-8] Toggle  ·  [↑/↓] Navigate  ·  [Enter] Confirm (min 2)  ·  [Esc] Cancel\033[0m")
		lines = append(lines, "")

		for i, m := range defaultDebateModelItems {
			prefix := "  "
			if i == cursor {
				prefix = "\033[1;36m> \033[0m"
			}

			dot := "\033[2m○\033[0m"
			nameColor := m.Name
			if selected[i] {
				dot = "\033[1;32m●\033[0m"
				nameColor = fmt.Sprintf("\033[1;32m%s\033[0m", m.Name)
			}

			line := fmt.Sprintf("%s%s [%d] %-36s \033[2m(%s)\033[0m", prefix, dot, i+1, nameColor, m.ID)
			lines = append(lines, line)
		}

		count := 0
		for _, v := range selected {
			if v {
				count++
			}
		}
		statusColor := "\033[2m"
		if count >= 2 {
			statusColor = "\033[1;32m"
		} else {
			statusColor = "\033[1;31m"
		}
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %sSelected: %d model(s) (Minimum 2 required)\033[0m", statusColor, count))

		if lastLinesCount > 0 {
			fmt.Print(fmt.Sprintf("\033[%dA", lastLinesCount))
		}
		for _, l := range lines {
			fmt.Print("\033[2K\r" + l + "\r\n")
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

		// Escape or 'q'
		if buf[0] == 27 && n == 1 || buf[0] == 'q' || buf[0] == 3 {
			if lastLinesCount > 0 {
				fmt.Print(fmt.Sprintf("\033[%dA", lastLinesCount))
				for i := 0; i < lastLinesCount; i++ {
					fmt.Print("\033[2K\r\n")
				}
				fmt.Print(fmt.Sprintf("\033[%dA", lastLinesCount))
			}
			return []string{"claude-sonnet-4-6", "deepseek-reasoner"}
		}

		// Arrow Keys: Esc [ A (Up), Esc [ B (Down)
		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			if buf[2] == 65 {
				if cursor > 0 {
					cursor--
				} else {
					cursor = len(defaultDebateModelItems) - 1
				}
				render()
				continue
			} else if buf[2] == 66 {
				if cursor < len(defaultDebateModelItems)-1 {
					cursor++
				} else {
					cursor = 0
				}
				render()
				continue
			}
		}

		if buf[0] == 'k' {
			if cursor > 0 {
				cursor--
			}
			render()
			continue
		}
		if buf[0] == 'j' {
			if cursor < len(defaultDebateModelItems)-1 {
				cursor++
			}
			render()
			continue
		}

		if buf[0] == ' ' {
			selected[cursor] = !selected[cursor]
			render()
			continue
		}

		if buf[0] >= '1' && buf[0] <= '8' {
			idx := int(buf[0] - '1')
			if idx < len(defaultDebateModelItems) {
				selected[idx] = !selected[idx]
				cursor = idx
				render()
			}
			continue
		}

		if buf[0] == '\r' || buf[0] == '\n' {
			var chosen []string
			for i, m := range defaultDebateModelItems {
				if selected[i] {
					chosen = append(chosen, m.ID)
				}
			}
			if len(chosen) >= 2 {
				if lastLinesCount > 0 {
					fmt.Print(fmt.Sprintf("\033[%dA", lastLinesCount))
					for i := 0; i < lastLinesCount; i++ {
						fmt.Print("\033[2K\r\n")
					}
					fmt.Print(fmt.Sprintf("\033[%dA", lastLinesCount))
				}
				return chosen
			}
		}
	}

	return []string{"claude-sonnet-4-6", "deepseek-reasoner"}
}

// pickDebateModel maps numeric index or slug to a valid model ID.
func pickDebateModel(input, fallback string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return fallback
	}
	for i, m := range defaultDebateModelItems {
		if input == fmt.Sprintf("%d", i+1) || strings.EqualFold(input, m.ID) || strings.EqualFold(input, m.Name) {
			return m.ID
		}
	}
	return input
}
