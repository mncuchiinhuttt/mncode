package ui

import (
	"bytes"
	"fmt"
	"mncode/pkg/agent"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// ReadInlinePrompt reads user prompt with chat frame, native scrolling & macOS editing shortcuts
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

	isProMax := s.Config.Workflow == "ultra-workflow" ||
		strings.EqualFold(s.Config.Effort, "pro max") ||
		strings.EqualFold(s.Config.Effort, "promax")

	render := func() {
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || width < 40 {
			width = 80
		}

		lines, dropdownCount, subagentCount, _, promptLen := buildPromptLines(s, input, cursorPos, selectedIdx, width, isProMax, branch)
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

		backLines := 2 + dropdownCount + subagentCount
		targetCol := promptLen + len([]rune(string(input[:cursorPos])))
		fmt.Printf("\r\033[%dA\033[%dC", backLines, targetCol)
	}

	RegisterCopyCallback(func() { render() })
	stopIdleWatcher := StartBrainrotIdleWatcher(s, func() { render() })
	defer stopIdleWatcher()

	render()

	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		ResetBrainrotActivity(func() { render() })
		rawStr := string(buf[:n])

		// 1. Image Paste via Ctrl+V
		if n == 1 && buf[0] == 22 {
			if imgPath, ok := TrySaveClipboardImage(s.WorkspaceDir); ok {
				relPath, _ := filepath.Rel(s.WorkspaceDir, imgPath)
				if relPath == "" {
					relPath = imgPath
				}
				tag := fmt.Sprintf("[Image: %s]", relPath)
				input = append(input[:cursorPos], append([]rune(tag), input[cursorPos:]...)...)
				cursorPos += len([]rune(tag))
				render()
				continue
			}
		}

		// 2. Multi-line Paste Collapsing
		if n > 1 && (bytes.Contains(buf[:n], []byte("\n")) || bytes.Contains(buf[:n], []byte("\r")) || n > 120) {
			pText := strings.ReplaceAll(rawStr, "\r\n", "\n")
			pText = strings.ReplaceAll(pText, "\r", "\n")
			pText = strings.Trim(pText, "\x1b[200~\x1b[201~")
			tag := StorePastedChunk(pText)
			input = append(input[:cursorPos], append([]rune(tag), input[cursorPos:]...)...)
			cursorPos += len([]rune(tag))
			selectedIdx = 0
			render()
			continue
		}

		// 3. Cmd+Backspace / Ctrl+U & Option+Backspace
		if buf[0] == 21 || (n == 2 && buf[0] == 27 && (buf[1] == 127 || buf[1] == 8)) || (n >= 4 && strings.Contains(rawStr, "3;2~")) {
			input, cursorPos = DeleteToStart(input, cursorPos)
			selectedIdx = 0
			render()
			continue
		}
		if buf[0] == 23 || (n == 2 && buf[0] == 27 && buf[1] == 'd') {
			input, cursorPos = DeleteWordBackward(input, cursorPos)
			selectedIdx = 0
			render()
			continue
		}

		// 4. Shift+Tab (Permission cycle)
		if (n == 3 && buf[0] == 27 && buf[1] == 91 && buf[2] == 90) || (n == 2 && buf[0] == 27 && buf[1] == 9) {
			cyclePermissionMode(s)
			render()
			continue
		}

		// 5. Arrow Keys & Navigation
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

		// 6. Option+Left / Option+Right
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

		// 7. Ctrl+A & Ctrl+E
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

		// 8. Ctrl+C / Ctrl+D (Exit)
		if n == 1 && (buf[0] == 3 || buf[0] == 4) {
			fmt.Print("\033[1A\r\033[J")
			PrintRizzGoodbye(s)
			return "", false
		}

		// 9. Enter & Tab
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
			finalInput := ResolvePastedContent(string(input))
			return finalInput, true
		}

		if n == 1 && buf[0] == 9 {
			newIn, newPos, handled, _ := ApplyDropdownSelection(s, input, cursorPos, selectedIdx, true)
			if handled {
				input, cursorPos, selectedIdx = newIn, newPos, 0
				render()
			}
			continue
		}

		if n == 1 && buf[0] == 27 {
			input, cursorPos, selectedIdx = nil, 0, 0
			render()
			continue
		}

		// 10. UTF-8 rune parsing
		i := 0
		for i < n {
			b := buf[i]
			if b == 27 {
				i++
				for i < n && (buf[i] < 0x40 || buf[i] > 0x7E) {
					i++
				}
				if i < n {
					i++
				}
				continue
			}

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

	finalInput := ResolvePastedContent(string(input))
	return finalInput, true
}
