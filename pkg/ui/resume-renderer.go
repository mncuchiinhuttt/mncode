package ui

import (
	"fmt"
	"mncode/pkg/provider"
	"strings"
)

func printRawLine(text string) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for _, l := range lines {
		fmt.Printf("\r\033[K%s\r\n", l)
	}
}

// RenderResumedHistory renders all messages from a restored session to terminal UI
func RenderResumedHistory(history []provider.Message) {
	if len(history) == 0 {
		return
	}
	t := GetCurrentTheme()
	printRawLine("")

	for _, m := range history {
		switch m.Role {
		case provider.RoleUser:
			if strings.TrimSpace(m.Content) == "" {
				continue
			}
			divider := GrayText(strings.Repeat("─", 60))
			printRawLine(divider)
			printRawLine(fmt.Sprintf("%s %s", BoldPastelPink("❯❯"), Colorize(AttrBold+t.Text, m.Content)))
			printRawLine("")

		case provider.RoleAssistant:
			if m.Thinking != "" {
				lines := strings.Split(strings.TrimSpace(m.Thinking), "\n")
				preview := lines[0]
				if len([]rune(preview)) > 60 {
					preview = string([]rune(preview)[:57]) + "…"
				}
				printRawLine(fmt.Sprintf("%s %s", Colorize(AttrBold+t.Secondary, "💭 [Thinking]"), GrayText(fmt.Sprintf("%s (%d lines)", preview, len(lines)))))
			}

			for _, tc := range m.ToolCalls {
				printRawLine(RenderToolCallFormatted(&tc))
			}

			if m.Content != "" {
				printRawLine(RenderMarkdown(m.Content))
			}

		case provider.RoleTool:
			for _, tr := range m.ToolResults {
				printRawLine(RenderToolResultFormatted(tr.Name, tr.Content, tr.IsError))
			}
		}
	}
	printRawLine("")
}
