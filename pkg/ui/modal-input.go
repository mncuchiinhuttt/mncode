package ui

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// ReadModalInput displays a clean, modal prompt box in raw terminal mode
// to collect user text (e.g. API keys, account names, custom model IDs)
// without falling back to buggy cooked-mode line readers.
func ReadModalInput(title, prompt, defaultValue string) (string, bool) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return defaultValue, true
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return defaultValue, false
	}
	defer func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Print("\033[?25h")
	}()

	input := []rune(defaultValue)
	cursorPos := len(input)
	linesRendered := 0

	render := func() {
		if linesRendered > 0 {
			fmt.Printf("\r\033[%dA\033[J", linesRendered)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s\r\n", BoldPastelPink("╭── ["+title+"] ────────────────────────────────────────╮")))
		sb.WriteString(fmt.Sprintf("│ %s\r\n", GrayText(prompt)))

		displayVal := string(input)
		if strings.Contains(strings.ToLower(title), "key") || strings.Contains(strings.ToLower(title), "token") {
			if len(input) > 8 {
				displayVal = string(input[:4]) + strings.Repeat("•", len(input)-8) + string(input[len(input)-4:])
			}
		}

		sb.WriteString(fmt.Sprintf("│ %s %s\r\n", BoldCyan("❯"), Bold(displayVal)))
		sb.WriteString(fmt.Sprintf("%s\r\n", BoldPastelPink("╰────────────────────────────────────────────────────────╯")))
		sb.WriteString(fmt.Sprintf("  %s\r\n", GrayText("(Press Enter to confirm, Esc to cancel)")))

		out := sb.String()
		linesRendered = strings.Count(out, "\n")
		fmt.Print(out)
	}

	render()

	buf := make([]byte, 256)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		// Esc, Ctrl+C, Ctrl+D -> Cancel
		if buf[0] == 3 || buf[0] == 4 || (n == 1 && buf[0] == 27) {
			if linesRendered > 0 {
				fmt.Printf("\r\033[%dA\033[J", linesRendered)
			}
			return "", false
		}

		// Enter -> Submit
		if buf[0] == 13 || buf[0] == 10 {
			if linesRendered > 0 {
				fmt.Printf("\r\033[%dA\033[J", linesRendered)
			}
			return strings.TrimSpace(string(input)), true
		}

		// Backspace / Delete
		if buf[0] == 127 || buf[0] == 8 {
			if cursorPos > 0 {
				input = append(input[:cursorPos-1], input[cursorPos:]...)
				cursorPos--
			}
			render()
			continue
		}

		// Arrow keys (ANSI escape sequence: ESC [ A/B/C/D)
		if n >= 3 && buf[0] == 27 && buf[1] == 91 {
			if buf[2] == 'D' && cursorPos > 0 { // Left
				cursorPos--
			} else if buf[2] == 'C' && cursorPos < len(input) { // Right
				cursorPos++
			}
			render()
			continue
		}

		// Printable runes & pasted chunks
		i := 0
		for i < n {
			if buf[i] == 13 || buf[i] == 10 {
				if linesRendered > 0 {
					fmt.Printf("\r\033[%dA\033[J", linesRendered)
				}
				return strings.TrimSpace(string(input)), true
			}
			if buf[i] >= 32 || buf[i] >= 0x80 {
				r, size := utf8.DecodeRune(buf[i:n])
				if r != utf8.RuneError && size > 0 {
					input = append(input[:cursorPos], append([]rune{r}, input[cursorPos:]...)...)
					cursorPos++
					i += size
					continue
				}
			}
			i++
		}
		render()
	}

	if linesRendered > 0 {
		fmt.Printf("\r\033[%dA\033[J", linesRendered)
	}
	return strings.TrimSpace(string(input)), true
}
