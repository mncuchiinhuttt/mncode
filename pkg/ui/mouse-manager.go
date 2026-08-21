package ui

import (
	"fmt"
	"strconv"
	"strings"
)

type MouseEvent struct {
	Button       int
	X            int
	Y            int
	IsPress      bool
	IsScrollUp   bool
	IsScrollDown bool
}

// ParseMouseEvent parses SGR mouse sequences \033[<btn;x;y;M or m
func ParseMouseEvent(str string) (*MouseEvent, bool) {
	if !strings.HasPrefix(str, "\033[<") {
		return nil, false
	}

	body := strings.TrimPrefix(str, "\033[<")
	if len(body) == 0 {
		return nil, false
	}

	action := body[len(body)-1]
	body = body[:len(body)-1]
	parts := strings.Split(body, ";")
	if len(parts) < 3 {
		return nil, false
	}

	btn, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, false
	}
	x, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, false
	}
	y, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, false
	}

	ev := &MouseEvent{
		Button:       btn,
		X:            x,
		Y:            y,
		IsPress:      (action == 'M'),
		IsScrollUp:   (btn == 64),
		IsScrollDown: (btn == 65),
	}
	return ev, true
}

// ParseCursorPosition parses DSR response \033[row;colR
func ParseCursorPosition(str string) (int, int, bool) {
	if !strings.HasPrefix(str, "\033[") || !strings.HasSuffix(str, "R") {
		return 0, 0, false
	}

	inner := strings.TrimSuffix(strings.TrimPrefix(str, "\033["), "R")
	parts := strings.Split(inner, ";")
	if len(parts) != 2 {
		return 0, 0, false
	}

	row, err1 := strconv.Atoi(parts[0])
	col, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return row, col, true
}

// EnableMouseTracking enables SGR mouse protocol in terminal
func EnableMouseTracking() {
	fmt.Print("\033[?1000h\033[?1006h")
}

// DisableMouseTracking disables SGR mouse protocol in terminal
func DisableMouseTracking() {
	fmt.Print("\033[?1000l\033[?1006l")
}

// ScrollTerminal scrolls terminal viewport up or down
func ScrollTerminal(up bool, lines int) {
	if lines <= 0 {
		lines = 3
	}
	if up {
		fmt.Printf("\033[%dS", lines)
	} else {
		fmt.Printf("\033[%dT", lines)
	}
}
