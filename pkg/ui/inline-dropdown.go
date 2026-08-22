package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"strings"
)

type DropdownItem struct {
	Primary   string
	Secondary string
	Icon      string
}

// GetActiveDropdownItems returns the items and category for the current input & cursor
func GetActiveDropdownItems(s *agent.Session, input []rune, cursorPos int) ([]DropdownItem, string, int) {
	inputStr := string(input)

	// 1. Check for @ mention
	if atToken, atIdx, hasAt := GetLastAtToken(input, cursorPos); hasAt {
		atOpts := GetMatchingAtOptions(s.WorkspaceDir, atToken)
		var items []DropdownItem
		for _, o := range atOpts {
			icon := "📄"
			if o.Type == "folder" {
				icon = "📁"
			} else if o.Type == "git" {
				icon = "🌿"
			} else if o.Type == "special" {
				icon = "🔍"
			}
			items = append(items, DropdownItem{
				Primary:   o.Tag,
				Secondary: o.Detail,
				Icon:      icon,
			})
		}
		return items, "at", atIdx
	}

	// 2. Check for / slash command (only show dropdown while typing command name)
	if strings.HasPrefix(inputStr, "/") && !strings.Contains(inputStr, " ") {
		slashOpts := getMatchingSlashOptions(s, inputStr)
		var items []DropdownItem
		for _, o := range slashOpts {
			items = append(items, DropdownItem{
				Primary:   o.Command,
				Secondary: o.Description,
				Icon:      "",
			})
		}
		return items, "slash", 0
	}

	return nil, "", 0
}

// GetLastAtToken returns the active @ token, start index, and true if active
func GetLastAtToken(input []rune, cursorPos int) (string, int, bool) {
	if cursorPos > len(input) {
		cursorPos = len(input)
	}
	sub := string(input[:cursorPos])
	atIdx := strings.LastIndex(sub, "@")
	if atIdx == -1 {
		return "", -1, false
	}
	token := sub[atIdx:]
	if strings.Contains(token, " ") {
		return "", -1, false
	}
	return token, atIdx, true
}

// RenderDropdownLines renders the active dropdown items within width
func RenderDropdownLines(items []DropdownItem, selectedIdx int, width int) ([]string, int) {
	if len(items) == 0 {
		return nil, 0
	}

	maxDisplay := 5
	if len(items) < maxDisplay {
		maxDisplay = len(items)
	}
	startIdx := 0
	if selectedIdx >= maxDisplay {
		startIdx = selectedIdx - maxDisplay + 1
	}
	endIdx := startIdx + maxDisplay
	if endIdx > len(items) {
		endIdx = len(items)
		startIdx = endIdx - maxDisplay
		if startIdx < 0 {
			startIdx = 0
		}
	}

	var lines []string
	colWidth := 28
	for i := startIdx; i < endIdx; i++ {
		item := items[i]
		primaryRunes := []rune(item.Primary)
		primaryStr := item.Primary
		if len(primaryRunes) > colWidth {
			primaryStr = string(primaryRunes[:colWidth-1]) + "…"
		} else {
			primaryStr = primaryStr + strings.Repeat(" ", colWidth-len(primaryRunes))
		}

		maxDescLen := width - 4 - colWidth - 6
		if maxDescLen < 0 {
			maxDescLen = 0
		}
		secStr := truncateText(item.Secondary, maxDescLen)

		prefixIcon := ""
		if item.Icon != "" {
			prefixIcon = item.Icon + " "
		}

		if i == selectedIdx {
			lines = append(lines, fmt.Sprintf("  \033[1;36m> %s%s\033[0m \033[90m%s\033[0m",
				prefixIcon, primaryStr, secStr))
		} else {
			lines = append(lines, fmt.Sprintf("    %s\033[36m%s\033[0m \033[2m%s\033[0m",
				prefixIcon, primaryStr, secStr))
		}
	}

	return lines, endIdx - startIdx
}
