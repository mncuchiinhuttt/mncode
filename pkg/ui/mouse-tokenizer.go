package ui

import (
	"fmt"
	"strconv"
	"strings"
)

type MouseAction struct {
	Button    int
	Col       int
	Row       int
	IsPress   bool
	IsRelease bool
	IsScroll  bool
	ScrollUp  bool
}

// ConsumeMouseSequence parses an SGR mouse sequence starting at buf[i]
// Returns (action, nextIndex, handled)
func ConsumeMouseSequence(buf []byte, i int, n int) (*MouseAction, int, bool) {
	if i+3 >= n || buf[i] != 27 || buf[i+1] != '[' || buf[i+2] != '<' {
		return nil, i, false
	}

	end := i + 3
	for end < n && buf[end] != 'M' && buf[end] != 'm' {
		end++
	}
	if end >= n {
		return nil, n, true // Incomplete at end of buffer, swallow to prevent text leak
	}

	termChar := buf[end]
	body := string(buf[i+3 : end])
	parts := strings.Split(body, ";")
	if len(parts) != 3 {
		return nil, end + 1, true
	}

	btn, err1 := strconv.Atoi(parts[0])
	col, err2 := strconv.Atoi(parts[1])
	row, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return nil, end + 1, true
	}

	act := &MouseAction{
		Button:    btn,
		Col:       col,
		Row:       row,
		IsPress:   (termChar == 'M'),
		IsRelease: (termChar == 'm'),
		IsScroll:  (btn == 64 || btn == 65),
		ScrollUp:  (btn == 64),
	}
	return act, end + 1, true
}

// ConsumeCursorReport parses DSR \033[row;colR at buf[i]
func ConsumeCursorReport(buf []byte, i int, n int) (int, int, int, bool) {
	if i+2 >= n || buf[i] != 27 || buf[i+1] != '[' {
		return 0, 0, i, false
	}

	end := i + 2
	for end < n && buf[end] != 'R' {
		if buf[end] < '0' && buf[end] != ';' && buf[end] != '?' {
			return 0, 0, i, false
		}
		end++
	}
	if end >= n {
		return 0, 0, i, false
	}

	body := string(buf[i+2 : end])
	parts := strings.Split(body, ";")
	if len(parts) != 2 {
		return 0, 0, i, false
	}

	row, err1 := strconv.Atoi(parts[0])
	col, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, i, false
	}
	return row, col, end + 1, true
}

// EnableMouseTracking enables SGR mouse protocol
func EnableMouseTracking() {
	fmt.Print("\033[?1000h\033[?1006h")
}

// DisableMouseTracking disables SGR mouse protocol
func DisableMouseTracking() {
	fmt.Print("\033[?1000l\033[?1006l")
}
