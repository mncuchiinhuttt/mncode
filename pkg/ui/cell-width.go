package ui

import (
	"regexp"
	"strings"
)

var ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\([B0-9]`)

// StripANSI removes all ANSI escape sequences from a string
func StripANSI(str string) string {
	return ansiEscapeRegex.ReplaceAllString(str, "")
}

// RuneCellWidth returns the visual column width of a single rune in a terminal
func RuneCellWidth(r rune) int {
	// Zero-width modifiers and control characters
	if r < 32 || (r >= 0x7F && r < 0xA0) {
		return 0
	}
	// Zero-width spaces, joiners, variation selectors
	if (r >= 0x200B && r <= 0x200F) || (r >= 0x2028 && r <= 0x202E) || (r >= 0x2060 && r <= 0x206F) || (r >= 0xFE00 && r <= 0xFE0F) {
		return 0
	}
	// Combining diacritical marks
	if r >= 0x0300 && r <= 0x036F {
		return 0
	}

	// Double-width CJK ideographs & symbols
	if (r >= 0x1100 && r <= 0x115F) ||
		(r >= 0x2E80 && r <= 0xA4CF && r != 0x303F) ||
		(r >= 0xAC00 && r <= 0xD7A3) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFE10 && r <= 0xFE19) ||
		(r >= 0xFE30 && r <= 0xFE6F) ||
		(r >= 0xFF00 && r <= 0xFF60) ||
		(r >= 0xFFE0 && r <= 0xFFE6) ||
		(r >= 0x20000 && r <= 0x2FFFD) ||
		(r >= 0x30000 && r <= 0x3FFFD) {
		return 2
	}

	// Double-width Emojis and miscellaneous symbols
	if (r >= 0x1F000 && r <= 0x1FAFF) ||
		(r >= 0x1F300 && r <= 0x1F5FF) ||
		(r >= 0x1F600 && r <= 0x1F64F) ||
		(r >= 0x1F680 && r <= 0x1F6FF) ||
		(r >= 0x2600 && r <= 0x27BF) ||
		(r >= 0x2300 && r <= 0x23FF) ||
		(r >= 0x2B50 && r <= 0x2B55) {
		return 2
	}

	return 1
}

// DisplayCellWidth returns the visual column count of a string taking ANSI codes and Emojis into account
func DisplayCellWidth(str string) int {
	clean := StripANSI(str)
	w := 0
	for _, r := range clean {
		w += RuneCellWidth(r)
	}
	return w
}

func visualLen(s string) int {
	return DisplayCellWidth(s)
}

// PadToCellWidth pads or trims a string with spaces until it exactly equals targetWidth columns
func PadToCellWidth(str string, targetWidth int) string {
	curW := DisplayCellWidth(str)
	if curW == targetWidth {
		return str
	}
	if curW > targetWidth {
		return TruncateToCellWidth(str, targetWidth)
	}
	return str + strings.Repeat(" ", targetWidth-curW)
}

// TruncateToCellWidth truncates a string with visual ellipsis "…" so its display width <= maxWidth
func TruncateToCellWidth(str string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if DisplayCellWidth(str) <= maxWidth {
		return str
	}

	clean := StripANSI(str)
	var cur []rune
	curW := 0
	for _, r := range clean {
		rw := RuneCellWidth(r)
		if curW+rw > maxWidth-1 {
			break
		}
		cur = append(cur, r)
		curW += rw
	}
	res := string(cur)
	for DisplayCellWidth(res+"…") > maxWidth && len(cur) > 0 {
		cur = cur[:len(cur)-1]
		res = string(cur)
	}
	return res + "…"
}
