package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// ReadInlinePrompt reads user prompt with chat frame, full UTF-8 & Vietnamese IME support
func ReadInlinePrompt(s *agent.Session) (string, bool) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return readPipedLine()
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return readPipedLine()
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	var input []rune
	cursorPos, selectedIdx := 0, 0
	renderedBefore := false
	branch := GetGitBranchOrFolder(s.WorkspaceDir)

	isProMax := s.Config.Workflow == "ultracode" ||
		strings.EqualFold(s.Config.Effort, "pro max") ||
		strings.EqualFold(s.Config.Effort, "promax")

	render := func() {
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || width < 40 {
			width = 80
		}

		lines, dropdownCount, _, promptLen := buildPromptLines(s, input, cursorPos, selectedIdx, width, isProMax, branch)

		if renderedBefore {
			fmt.Print("\033[1A\r\033[J")
		}
		renderedBefore = true

		for i, line := range lines {
			if i < len(lines)-1 {
				fmt.Printf("\r\033[K%s\r\n", line)
			} else {
				fmt.Printf("\r\033[K%s", line)
			}
		}

		backLines := 2 + dropdownCount
		targetCol := promptLen + len([]rune(string(input[:cursorPos])))
		fmt.Printf("\r\033[%dA\033[%dC", backLines, targetCol)
	}

	RegisterCopyCallback(func() { render() })
	stopIdleWatcher := StartBrainrotIdleWatcher(s, func() { render() })
	defer stopIdleWatcher()

	render()

	buf := make([]byte, 128)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		ResetBrainrotActivity(func() { render() })

		// Shift+Tab permission cycle
		if (n == 3 && buf[0] == 27 && buf[1] == 91 && buf[2] == 90) || (n == 2 && buf[0] == 27 && buf[1] == 9) {
			cyclePermissionMode(s)
			render()
			continue
		}

		// Escape sequences (Arrows)
		if n >= 3 && buf[0] == 27 && buf[1] == 91 {
			items, _, _ := GetActiveDropdownItems(s, input, cursorPos)
			switch buf[2] {
			case 'A':
				if len(items) > 0 && selectedIdx > 0 {
					selectedIdx--
					render()
				}
				continue
			case 'B':
				if len(items) > 0 && selectedIdx < len(items)-1 {
					selectedIdx++
					render()
				}
				continue
			case 'C':
				if cursorPos < len(input) {
					cursorPos++
					selectedIdx = 0
					render()
				}
				continue
			case 'D':
				if cursorPos > 0 {
					cursorPos--
					selectedIdx = 0
					render()
				} else if len(input) == 0 {
					oldState = openSubagentMonitor(s, oldState)
					renderedBefore = false
					render()
				}
				continue
			}
		}

		// Ctrl+A
		if n == 1 && buf[0] == 1 {
			oldState = openSubagentMonitor(s, oldState)
			renderedBefore = false
			render()
			continue
		}

		// Ctrl+C / Ctrl+D
		if n == 1 && (buf[0] == 3 || buf[0] == 4) {
			fmt.Print("\033[1A\r\033[J")
			PrintRizzGoodbye(s)
			return "", false
		}

		// Enter
		if n == 1 && (buf[0] == 13 || buf[0] == 10) {
			items, cat, atIdx := GetActiveDropdownItems(s, input, cursorPos)
			if len(items) > 0 && selectedIdx < len(items) {
				chosen := items[selectedIdx]
				if cat == "at" {
					prefix := string(input[:atIdx])
					suffix := string(input[cursorPos:])
					newStr := prefix + chosen.Primary + " " + suffix
					input = []rune(newStr)
					cursorPos = atIdx + len([]rune(chosen.Primary)) + 1
					selectedIdx = 0
					render()
					continue
				} else if cat == "slash" {
					if strings.HasPrefix(chosen.Primary, "/ck:") || chosen.Primary == "/btw" || chosen.Primary == "/model" {
						input = []rune(chosen.Primary + " ")
						cursorPos = len(input)
						selectedIdx = 0
						render()
						continue
					}
					input = []rune(chosen.Primary)
				}
			}
			promptSymbol := BoldCyan("❯")
			if isProMax {
				promptSymbol = "\033[1;38;5;218m❯\033[1;38;5;212m❯\033[0m"
			}
			fmt.Printf("\033[1A\r\033[J%s %s\r\n", promptSymbol, highlightPromptInput(string(input)))
			return string(input), true
		}

		// Tab
		if n == 1 && buf[0] == 9 {
			items, cat, atIdx := GetActiveDropdownItems(s, input, cursorPos)
			if len(items) > 0 && selectedIdx < len(items) {
				chosen := items[selectedIdx]
				if cat == "at" {
					prefix := string(input[:atIdx])
					suffix := string(input[cursorPos:])
					newStr := prefix + chosen.Primary + " " + suffix
					input = []rune(newStr)
					cursorPos = atIdx + len([]rune(chosen.Primary)) + 1
					selectedIdx = 0
					render()
				} else if cat == "slash" {
					input = []rune(chosen.Primary + " ")
					cursorPos = len(input)
					selectedIdx = 0
					render()
				}
			}
			continue
		}

		// Single Escape
		if n == 1 && buf[0] == 27 {
			input = nil
			cursorPos = 0
			render()
			continue
		}

		// Parse sequential characters (Multi-byte UTF-8, Vietnamese IME & mixed backspaces)
		i := 0
		for i < n {
			b := buf[i]
			if b == 127 || b == 8 {
				if cursorPos > 0 {
					input = append(input[:cursorPos-1], input[cursorPos:]...)
					cursorPos--
				}
				i++
			} else if b >= 32 || b >= 0x80 {
				r, size := utf8.DecodeRune(buf[i:n])
				if r != utf8.RuneError && size > 0 {
					input = append(input[:cursorPos], append([]rune{r}, input[cursorPos:]...)...)
					cursorPos++
					i += size
				} else {
					i++
				}
			} else {
				i++
			}
		}
		selectedIdx = 0
		render()
	}

	return string(input), true
}
