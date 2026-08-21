package ui

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	pasteStoreMu sync.Mutex
	pasteStore   = make(map[string]string)
	pasteCounter = 0
	pasteTagRe   = regexp.MustCompile(`\[Pasted (\d+) (lines|chars|KB)\]`)
)

// StorePastedChunk stores multi-line or long pasted text and returns a compact placeholder tag
func StorePastedChunk(text string) string {
	pasteStoreMu.Lock()
	defer pasteStoreMu.Unlock()

	pasteCounter++
	id := fmt.Sprintf("paste-%d", pasteCounter)
	pasteStore[id] = text

	lines := strings.Split(text, "\n")
	tag := ""
	if len(lines) > 2 {
		tag = fmt.Sprintf("[Pasted %d lines]", len(lines))
	} else if len(text) > 100 {
		tag = fmt.Sprintf("[Pasted %.1f KB]", float64(len(text))/1024.0)
	} else {
		return text
	}

	// Register tag mapping
	pasteStore[tag] = text
	return tag
}

// ResolvePastedContent expands all [Pasted ...] tokens back into their original full text
func ResolvePastedContent(input string) string {
	pasteStoreMu.Lock()
	defer pasteStoreMu.Unlock()

	result := input
	for tag, fullText := range pasteStore {
		if strings.Contains(result, tag) {
			result = strings.ReplaceAll(result, tag, fullText)
		}
	}
	return result
}

// FormatPastedTagHighlight renders [Pasted ...] tokens with a stylish highlighted pill
func FormatPastedTagHighlight(input string) string {
	return pasteTagRe.ReplaceAllStringFunc(input, func(m string) string {
		return fmt.Sprintf("\033[1;38;5;218m%s\033[0m", m)
	})
}
