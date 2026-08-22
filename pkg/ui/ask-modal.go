package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// AskQuestionOption represents a selectable option
type AskQuestionOption struct {
	Text        string
	IsCustom    bool
	IsChatMore  bool
	IsSelected  bool
}

// AskQuestionParams represents parameters for an interactive agent question
type AskQuestionParams struct {
	Question      string
	Options       []string
	IsMultiSelect bool
}

// PromptAgentQuestion opens an interactive multiple-choice & custom write-in modal
func PromptAgentQuestion(params AskQuestionParams) string {
	question := strings.TrimSpace(params.Question)
	if question == "" {
		question = "The agent has a question for you:"
	}

	var options []AskQuestionOption
	for _, opt := range params.Options {
		trimmed := strings.TrimSpace(opt)
		if trimmed != "" {
			options = append(options, AskQuestionOption{Text: trimmed})
		}
	}

	// Always append Write-in & Chat options
	options = append(options, AskQuestionOption{
		Text:     "💬 Other / Type custom answer...",
		IsCustom: true,
	})
	options = append(options, AskQuestionOption{
		Text:       "💭 Chat more about this / give extra context...",
		IsChatMore: true,
	})

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return promptSimpleQuestion(question, options)
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return promptSimpleQuestion(question, options)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	currentIdx := 0
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
		contentWidth := cardWidth - 4

		var lines []string
		lines = append(lines, "")

		title := "❓ Human-in-the-Loop · Question from Agent"
		dashes := cardWidth - DisplayCellWidth(title) - 8
		if dashes < 2 {
			dashes = 2
		}

		topHeader := fmt.Sprintf("%s %s %s",
			BoldCyan("╭── ["), Bold(title),
			BoldCyan("] "+strings.Repeat("─", dashes)+"╮"))
		lines = append(lines, topHeader)

		// Question wrap
		qLines := wrapText(question, contentWidth)
		for _, ql := range qLines {
			lines = append(lines, fmt.Sprintf("│ %s │", PadToCellWidth(Bold(ql), contentWidth)))
		}

		sep := fmt.Sprintf("├%s┤", strings.Repeat("─", cardWidth-2))
		lines = append(lines, sep)

		for i, opt := range options {
			marker := "  "
			numBadge := fmt.Sprintf("[%d]", i+1)
			if i == currentIdx {
				marker = BoldPastelPink("❯ ")
				numBadge = BoldPastelPink(fmt.Sprintf("[%d]", i+1))
			}

			optText := opt.Text
			if i == currentIdx {
				optText = Bold(opt.Text)
			}
			if opt.IsCustom || opt.IsChatMore {
				optText = Colorize(AttrDim, optText)
				if i == currentIdx {
					optText = BoldYellow(opt.Text)
				}
			}

			rowStr := fmt.Sprintf("%s%s %s", marker, numBadge, optText)
			lines = append(lines, fmt.Sprintf("│ %s │", PadToCellWidth(rowStr, contentWidth)))
		}

		bottomBorder := fmt.Sprintf("╰%s╯", strings.Repeat("─", cardWidth-2))
		footer := "  [↑/↓] Navigate · [1-9] Quick Select · [Enter] Choose · [Esc] Skip"
		lines = append(lines, bottomBorder, GrayText(footer))

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

		// Arrow Up / Down
		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A': // Up
				if currentIdx > 0 {
					currentIdx--
					render()
				}
				continue
			case 'B': // Down
				if currentIdx < len(options)-1 {
					currentIdx++
					render()
				}
				continue
			}
		}

		// Quick Digit Keys 1..9
		if n == 1 && buf[0] >= '1' && buf[0] <= '9' {
			idx := int(buf[0] - '1')
			if idx < len(options) {
				currentIdx = idx
				render()
				// Submit selection immediately
				selected := options[currentIdx]
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				fmt.Print("\n\r")
				if selected.IsCustom || selected.IsChatMore {
					return readCustomInput(selected.Text)
				}
				return fmt.Sprintf("User selected: %s", selected.Text)
			}
		}

		// Enter Key
		if buf[0] == '\r' || buf[0] == '\n' {
			selected := options[currentIdx]
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
			fmt.Print("\n\r")
			if selected.IsCustom || selected.IsChatMore {
				return readCustomInput(selected.Text)
			}
			return fmt.Sprintf("User selected: %s", selected.Text)
		}

		// Escape / Ctrl+C / q
		if buf[0] == 27 || buf[0] == 3 || buf[0] == 'q' || buf[0] == 'Q' {
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
			fmt.Print("\n\r")
			return "User skipped this question."
		}
	}

	return "User provided no response."
}

func readCustomInput(promptLabel string) string {
	fmt.Printf("\n%s %s\n%s ", BoldCyan("✏️"), Bold(promptLabel), BoldPastelPink("❯❯"))
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return "User provided no additional details."
	}
	return fmt.Sprintf("User write-in response: %s", line)
}

func promptSimpleQuestion(question string, options []AskQuestionOption) string {
	fmt.Printf("\n[Question]: %s\n", question)
	for i, opt := range options {
		fmt.Printf("  [%d] %s\n", i+1, opt.Text)
	}
	fmt.Print("\nEnter choice (or type custom text): ")
	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(ans)
	if num, err := strconv.Atoi(ans); err == nil && num >= 1 && num <= len(options) {
		selected := options[num-1]
		if selected.IsCustom || selected.IsChatMore {
			return readCustomInput(selected.Text)
		}
		return fmt.Sprintf("User selected: %s", selected.Text)
	}
	if ans != "" {
		return fmt.Sprintf("User response: %s", ans)
	}
	return "User skipped question."
}

func wrapText(text string, maxLen int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if DisplayCellWidth(cur.String()+" "+w) > maxLen {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(w)
		} else {
			if cur.Len() > 0 {
				cur.WriteString(" ")
			}
			cur.WriteString(w)
		}
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}
