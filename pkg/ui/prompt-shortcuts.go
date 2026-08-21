package ui

import (
	"mncode/pkg/agent"
	"strconv"
	"strings"
)

// ParseMouseClick returns target character index if a mouse click event is received
func ParseMouseClick(buf []byte, promptLen int, inputLen int) (int, bool) {
	str := string(buf)
	if !strings.HasPrefix(str, "\033[<") {
		return 0, false
	}

	body := strings.TrimPrefix(str, "\033[<")
	if len(body) == 0 {
		return 0, false
	}

	lastChar := body[len(body)-1]
	if lastChar != 'M' { // Only handle press
		return 0, true
	}

	body = body[:len(body)-1]
	parts := strings.Split(body, ";")
	if len(parts) < 3 {
		return 0, false
	}

	btn, err := strconv.Atoi(parts[0])
	if err != nil || btn != 0 {
		return 0, true
	}

	x, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false
	}

	col := x - promptLen - 1
	if col < 0 {
		col = 0
	}
	if col > inputLen {
		col = inputLen
	}
	return col, true
}

// DeleteToStart deletes from cursorPos to 0 (Cmd+Backspace / Ctrl+U)
func DeleteToStart(input []rune, cursorPos int) ([]rune, int) {
	if cursorPos <= 0 {
		return input, 0
	}
	return input[cursorPos:], 0
}

// DeleteWordBackward deletes the word preceding cursorPos (Option+Backspace / Ctrl+W)
func DeleteWordBackward(input []rune, cursorPos int) ([]rune, int) {
	if cursorPos <= 0 {
		return input, 0
	}
	newPos := cursorPos - 1
	for newPos > 0 && input[newPos] == ' ' {
		newPos--
	}
	for newPos > 0 && input[newPos-1] != ' ' {
		newPos--
	}
	res := append(input[:newPos], input[cursorPos:]...)
	return res, newPos
}

// MoveWordBackward moves cursorPos one word left (Option+Left)
func MoveWordBackward(input []rune, cursorPos int) int {
	if cursorPos <= 0 {
		return 0
	}
	pos := cursorPos - 1
	for pos > 0 && input[pos] == ' ' {
		pos--
	}
	for pos > 0 && input[pos-1] != ' ' {
		pos--
	}
	return pos
}

// MoveWordForward moves cursorPos one word right (Option+Right)
func MoveWordForward(input []rune, cursorPos int) int {
	length := len(input)
	if cursorPos >= length {
		return length
	}
	pos := cursorPos + 1
	for pos < length && input[pos] != ' ' {
		pos++
	}
	for pos < length && input[pos] == ' ' {
		pos++
	}
	return pos
}

// ApplyDropdownSelection applies autocompletion for @ and / dropdown items
func ApplyDropdownSelection(s *agent.Session, input []rune, cursorPos int, selectedIdx int, isTab bool) ([]rune, int, bool, bool) {
	items, cat, atIdx := GetActiveDropdownItems(s, input, cursorPos)
	if len(items) == 0 || selectedIdx >= len(items) {
		return input, cursorPos, false, false
	}

	chosen := items[selectedIdx]
	if cat == "at" {
		prefix := string(input[:atIdx])
		suffix := string(input[cursorPos:])
		newStr := prefix + chosen.Primary + " " + suffix
		newInput := []rune(newStr)
		newPos := atIdx + len([]rune(chosen.Primary)) + 1
		return newInput, newPos, true, true
	}

	if isTab || strings.HasPrefix(chosen.Primary, "/ck:") || chosen.Primary == "/btw" || chosen.Primary == "/model" {
		newInput := []rune(chosen.Primary + " ")
		return newInput, len(newInput), true, true
	}

	return []rune(chosen.Primary), len([]rune(chosen.Primary)), true, false
}
