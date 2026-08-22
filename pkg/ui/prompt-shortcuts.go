package ui

import (
	"mncode/pkg/agent"
)

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

	// For TAB key: Autocomplete the command with a trailing space and stay in prompt
	if isTab {
		newInput := []rune(chosen.Primary + " ")
		return newInput, len(newInput), true, true
	}

	// For ENTER key:
	// If it's "/btw", add space and let user type the message
	if chosen.Primary == "/btw" {
		newInput := []rune(chosen.Primary + " ")
		return newInput, len(newInput), true, true
	}

	// For all other slash commands and skills (/model, /theme, /status, /cook, /plan, /agents, /mcp, etc.):
	// Immediately execute the chosen command
	return []rune(chosen.Primary), len([]rune(chosen.Primary)), true, false
}
