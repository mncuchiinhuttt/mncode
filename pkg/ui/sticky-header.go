package ui

import (
	"fmt"
	"mncode/pkg/agent"
)

var currentStickyCleanup func()

// RenderStickyHeader paints the header card
func RenderStickyHeader(s *agent.Session) {
	PrintHeaderCard(s)
}

// ResetStickyHeader restores default full-terminal scrolling
func ResetStickyHeader() {
	if currentStickyCleanup != nil {
		currentStickyCleanup()
	} else {
		fmt.Print("\033[r")
	}
}
