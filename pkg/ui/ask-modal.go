package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"mncode/pkg/remote"

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
	IsBrainrot    bool
}

// PromptAgentQuestion opens an interactive multiple-choice & custom write-in modal
func PromptAgentQuestion(params AskQuestionParams) string {
	if rm := remote.GetGlobalRemote(); rm != nil && rm.IsActive {
		rm.PushQuestion(remote.QuestionPayload{
			Question:      params.Question,
			Options:       params.Options,
			IsMultiSelect: params.IsMultiSelect,
			IsBrainrot:    params.IsBrainrot,
		})
		defer rm.PushQuestionResolved()
	}

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

	customLabel := "💬 Other / Type custom answer..."
	chatLabel := "💭 Chat more about this / give extra context..."
	title := "⚡ Need your action · Agent Decision"
	footer := "  [↑/↓] Navigate · [1-9] Quick Select · [Enter] Choose · [Esc] Skip"

	if params.IsMultiSelect {
		title = "⚡ Need your action · Select Options (Multi-choice)"
		footer = "  [Space/1-9] Toggle · [↑/↓] Navigate · [Enter] Confirm · [Esc] Skip"
	}

	if params.IsBrainrot {
		if params.IsMultiSelect {
			title = "🔥 Yo Sigma! Pick all vibes that apply 🧠"
			footer = "  [Space/1-9] Toggle vibe · [↑/↓] Navigate · [Enter] Lock in all · [Esc] Nah skip"
		} else {
			title = "🔥 Yo Sigma! Lock in & pick a vibe 🧠"
			footer = "  [↑/↓] Navigate · [1-9] Quick Rizz · [Enter] Lock in · [Esc] Nah skip"
		}
		customLabel = "💬 Other / Cook custom rizz response..."
		chatLabel = "💭 Yap more about this / drop extra lore..."
	}

	// Always append Write-in & Chat options
	options = append(options, AskQuestionOption{
		Text:     customLabel,
		IsCustom: true,
	})
	options = append(options, AskQuestionOption{
		Text:       chatLabel,
		IsChatMore: true,
	})

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return promptSimpleQuestion(question, options, params.IsMultiSelect)
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return promptSimpleQuestion(question, options, params.IsMultiSelect)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	currentIdx := 0
	lastLinesCount := 0

	countSelected := func() int {
		c := 0
		for _, o := range options {
			if o.IsSelected && !o.IsCustom && !o.IsChatMore {
				c++
			}
		}
		return c
	}

	totalNavRows := len(options)
	if params.IsMultiSelect {
		totalNavRows++ // extra submit button row at bottom
	}

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
			if i == currentIdx {
				marker = BoldPastelPink("❯ ")
			}

			var prefixBadge string
			if params.IsMultiSelect {
				if opt.IsCustom || opt.IsChatMore {
					numBadge := fmt.Sprintf("[%d]", i+1)
					if i == currentIdx {
						numBadge = BoldPastelPink(fmt.Sprintf("[%d]", i+1))
					}
					prefixBadge = numBadge
				} else {
					checkMark := GrayText("☐")
					if opt.IsSelected {
						checkMark = BoldGreen("☑")
					}
					numBadge := fmt.Sprintf("[%d]", i+1)
					if i == currentIdx {
						numBadge = BoldPastelPink(fmt.Sprintf("[%d]", i+1))
					}
					prefixBadge = fmt.Sprintf("%s %s", numBadge, checkMark)
				}
			} else {
				numBadge := fmt.Sprintf("[%d]", i+1)
				if i == currentIdx {
					numBadge = BoldPastelPink(fmt.Sprintf("[%d]", i+1))
				}
				prefixBadge = numBadge
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

			rowStr := fmt.Sprintf("%s%s %s", marker, prefixBadge, optText)
			lines = append(lines, fmt.Sprintf("│ %s │", PadToCellWidth(rowStr, contentWidth)))
		}

		// Multi-select Submit row
		if params.IsMultiSelect {
			sepMid := fmt.Sprintf("│ %s │", PadToCellWidth(GrayText(strings.Repeat("─", 20)), contentWidth))
			lines = append(lines, sepMid)

			submitMarker := "  "
			if currentIdx == len(options) {
				submitMarker = BoldPastelPink("❯ ")
			}
			submitLabel := fmt.Sprintf("🟢 [Enter] Confirm & Submit (%d selected)", countSelected())
			if params.IsBrainrot {
				submitLabel = fmt.Sprintf("🔥 [Enter] Lock in selection (%d vibes picked)", countSelected())
			}
			if currentIdx == len(options) {
				submitLabel = BoldGreen(submitLabel)
			} else {
				submitLabel = GrayText(submitLabel)
			}
			lines = append(lines, fmt.Sprintf("│ %s │", PadToCellWidth(submitMarker+submitLabel, contentWidth)))
		}

		bottomBorder := fmt.Sprintf("╰%s╯", strings.Repeat("─", cardWidth-2))
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

	collectMultiResults := func(extraWriteIn string) string {
		var selectedTexts []string
		for _, o := range options {
			if o.IsSelected && !o.IsCustom && !o.IsChatMore {
				selectedTexts = append(selectedTexts, o.Text)
			}
		}
		if extraWriteIn != "" {
			selectedTexts = append(selectedTexts, extraWriteIn)
		}
		if len(selectedTexts) == 0 && currentIdx < len(options) {
			o := options[currentIdx]
			if !o.IsCustom && !o.IsChatMore {
				selectedTexts = append(selectedTexts, o.Text)
			}
		}
		if len(selectedTexts) == 0 {
			return "User selected no options."
		}
		if len(selectedTexts) == 1 {
			return fmt.Sprintf("User selected: %s", selectedTexts[0])
		}
		return fmt.Sprintf("User selected (multi-select): [\"%s\"]", strings.Join(selectedTexts, "\", \""))
	}

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
				if currentIdx < totalNavRows-1 {
					currentIdx++
					render()
				}
				continue
			}
		}

		// Space Key -> Toggle checkbox in Multi-Select
		if buf[0] == ' ' && params.IsMultiSelect {
			if currentIdx < len(options) {
				opt := &options[currentIdx]
				if !opt.IsCustom && !opt.IsChatMore {
					opt.IsSelected = !opt.IsSelected
					render()
					continue
				}
			}
		}

		// Quick Digit Keys 1..9
		if n == 1 && buf[0] >= '1' && buf[0] <= '9' {
			idx := int(buf[0] - '1')
			if idx < len(options) {
				if params.IsMultiSelect {
					// In multi-select: digit toggles checkbox
					opt := &options[idx]
					if !opt.IsCustom && !opt.IsChatMore {
						opt.IsSelected = !opt.IsSelected
						currentIdx = idx
						render()
						continue
					} else {
						// Custom write-in
						currentIdx = idx
						render()
						_ = term.Restore(int(os.Stdin.Fd()), oldState)
						fmt.Print("\n\r")
						writeIn := readCustomInput(opt.Text)
						return collectMultiResults(writeIn)
					}
				} else {
					// Single-select: immediately choose
					currentIdx = idx
					render()
					selected := options[currentIdx]
					_ = term.Restore(int(os.Stdin.Fd()), oldState)
					fmt.Print("\n\r")
					if selected.IsCustom || selected.IsChatMore {
						return readCustomInput(selected.Text)
					}
					return fmt.Sprintf("User selected: %s", selected.Text)
				}
			}
		}

		// Enter Key
		if buf[0] == '\r' || buf[0] == '\n' {
			if params.IsMultiSelect {
				// If on write-in option
				if currentIdx < len(options) && (options[currentIdx].IsCustom || options[currentIdx].IsChatMore) {
					selected := options[currentIdx]
					_ = term.Restore(int(os.Stdin.Fd()), oldState)
					fmt.Print("\n\r")
					writeIn := readCustomInput(selected.Text)
					return collectMultiResults(writeIn)
				}

				// If on regular option or submit row -> submit all checked
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				fmt.Print("\n\r")
				return collectMultiResults("")
			} else {
				// Single select
				selected := options[currentIdx]
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				fmt.Print("\n\r")
				if selected.IsCustom || selected.IsChatMore {
					return readCustomInput(selected.Text)
				}
				return fmt.Sprintf("User selected: %s", selected.Text)
			}
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

func promptSimpleQuestion(question string, options []AskQuestionOption, isMultiSelect bool) string {
	fmt.Printf("\n[Question]: %s\n", question)
	for i, opt := range options {
		fmt.Printf("  [%d] %s\n", i+1, opt.Text)
	}
	if isMultiSelect {
		fmt.Print("\nEnter choices separated by commas (e.g. 1, 3) or type custom text: ")
	} else {
		fmt.Print("\nEnter choice (or type custom text): ")
	}

	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(ans)

	if isMultiSelect {
		parts := strings.Split(ans, ",")
		var selected []string
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if num, err := strconv.Atoi(trimmed); err == nil && num >= 1 && num <= len(options) {
				opt := options[num-1]
				if !opt.IsCustom && !opt.IsChatMore {
					selected = append(selected, opt.Text)
				}
			}
		}
		if len(selected) > 0 {
			return fmt.Sprintf("User selected (multi-select): [\"%s\"]", strings.Join(selected, "\", \""))
		}
	} else {
		if num, err := strconv.Atoi(ans); err == nil && num >= 1 && num <= len(options) {
			selected := options[num-1]
			if selected.IsCustom || selected.IsChatMore {
				return readCustomInput(selected.Text)
			}
			return fmt.Sprintf("User selected: %s", selected.Text)
		}
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
