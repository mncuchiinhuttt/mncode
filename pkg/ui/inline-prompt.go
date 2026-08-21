package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// ReadInlinePrompt reads user prompt with chat frame, mouse click navigation & smooth scrolling
func ReadInlinePrompt(s *agent.Session) (string, bool) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return readPipedLine()
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return readPipedLine()
	}
	defer func() {
		DisableMouseTracking()
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}()

	EnableMouseTracking()

	var input []rune
	cursorPos, selectedIdx := 0, 0
	promptRow := 0
	renderedBefore := false
	branch := GetGitBranchOrFolder(s.WorkspaceDir)

	isProMax := s.Config.Workflow == "ultracode" ||
		strings.EqualFold(s.Config.Effort, "pro max") ||
		strings.EqualFold(s.Config.Effort, "promax")

	promptLen := 2
	if isProMax {
		promptLen = 3
	}

	render := func() {
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || width < 40 {
			width = 80
		}

		lines, dropdownCount, _, pLen := buildPromptLines(s, input, cursorPos, selectedIdx, width, isProMax, branch)
		promptLen = pLen

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
		fmt.Printf("\r\033[%dA\033[%dC\033[6n", backLines, targetCol)
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
		rawStr := string(buf[:n])

		// 1. DSR Cursor Position Response (\033[row;colR)
		if r, _, ok := ParseCursorPosition(rawStr); ok {
			promptRow = r
			continue
		}

		// 2. Mouse Click & Scrolling
		if ev, ok := ParseMouseEvent(rawStr); ok {
			if ev.IsScrollUp {
				ScrollTerminal(true, 3)
				continue
			}
			if ev.IsScrollDown {
				ScrollTerminal(false, 3)
				continue
			}
			if ev.IsPress && ev.Button == 0 {
				if promptRow > 0 && ev.Y != promptRow {
					continue // Clicked outside prompt line, do not move cursor
				}
				target := ev.X - promptLen - 1
				if target < 0 {
					target = 0
				}
				if target > len(input) {
					target = len(input)
				}
				cursorPos = target
				selectedIdx = 0
				render()
				continue
			}
			continue
		}

		// 3. Cmd+Backspace / Ctrl+U (Delete to start of line)
		if buf[0] == 21 || (n == 2 && buf[0] == 27 && (buf[1] == 127 || buf[1] == 8)) || (n >= 4 && strings.Contains(rawStr, "3;2~")) {
			input, cursorPos = DeleteToStart(input, cursorPos)
			selectedIdx = 0
			render()
			continue
		}

		// 4. Option+Backspace / Ctrl+W (Delete word backwards)
		if buf[0] == 23 || (n == 2 && buf[0] == 27 && buf[1] == 'd') {
			input, cursorPos = DeleteWordBackward(input, cursorPos)
			selectedIdx = 0
			render()
			continue
		}

		// 5. Shift+Tab (Permission cycle)
		if (n == 3 && buf[0] == 27 && buf[1] == 91 && buf[2] == 90) || (n == 2 && buf[0] == 27 && buf[1] == 9) {
			cyclePermissionMode(s)
			render()
			continue
		}

		// 6. Arrow Keys & Navigation
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
			case 'H':
				cursorPos = 0
				render()
				continue
			case 'F':
				cursorPos = len(input)
				render()
				continue
			}
		}

		// 7. Option+Left / Option+Right (Word jump)
		if n == 2 && buf[0] == 27 && buf[1] == 'b' {
			cursorPos = MoveWordBackward(input, cursorPos)
			render()
			continue
		}
		if n == 2 && buf[0] == 27 && buf[1] == 'f' {
			cursorPos = MoveWordForward(input, cursorPos)
			render()
			continue
		}

		// 8. Ctrl+A (Line start / Monitor) & Ctrl+E (Line end)
		if n == 1 && buf[0] == 1 {
			if len(input) == 0 {
				oldState = openSubagentMonitor(s, oldState)
				renderedBefore = false
			} else {
				cursorPos = 0
			}
			render()
			continue
		}
		if n == 1 && buf[0] == 5 {
			cursorPos = len(input)
			render()
			continue
		}

		// 9. Ctrl+C / Ctrl+D (Exit)
		if n == 1 && (buf[0] == 3 || buf[0] == 4) {
			fmt.Print("\033[1A\r\033[J")
			PrintRizzGoodbye(s)
			return "", false
		}

		// 10. Enter & Tab Autocomplete
		if n == 1 && (buf[0] == 13 || buf[0] == 10) {
			newIn, newPos, handled, continueLoop := ApplyDropdownSelection(s, input, cursorPos, selectedIdx, false)
			if handled {
				input, cursorPos, selectedIdx = newIn, newPos, 0
				if continueLoop {
					render()
					continue
				}
			}
			promptSymbol := BoldCyan("❯")
			if isProMax {
				promptSymbol = "\033[1;38;5;218m❯\033[1;38;5;212m❯\033[0m"
			}
			fmt.Printf("\033[1A\r\033[J%s %s\r\n", promptSymbol, highlightPromptInput(string(input)))
			return string(input), true
		}

		if n == 1 && buf[0] == 9 {
			newIn, newPos, handled, _ := ApplyDropdownSelection(s, input, cursorPos, selectedIdx, true)
			if handled {
				input, cursorPos, selectedIdx = newIn, newPos, 0
				render()
			}
			continue
		}

		// 11. Escape
		if n == 1 && buf[0] == 27 {
			input, cursorPos, selectedIdx = nil, 0, 0
			render()
			continue
		}

		// 12. Parse sequential characters & multi-byte UTF-8
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
