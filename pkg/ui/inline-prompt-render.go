package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"strings"
)

func buildPromptLines(s *agent.Session, input []rune, cursorPos int, selectedIdx int, width int, isProMax bool, branch string) ([]string, int, string, int) {
	if s.Config.GetSetting("show_branch_name", "true") == "false" {
		branch = ""
	}
	var promptPrefix string
	promptLen := 2

	if isProMax {
		promptPrefix = "\033[1;38;5;218m❯\033[1;38;5;212m❯\033[0m"
		promptLen = 3
	} else {
		promptPrefix = BoldCyan("❯")
		promptLen = 2
	}

	modelName := s.Config.Model
	if modelName == "" {
		modelName = "gemini-3.7-flash-high"
	}

	var bottomBorder string
	if isProMax {
		titleLen := len([]rune(modelName)) + 3 + 14
		dashesLen := width - titleLen - 4
		if dashesLen < 2 {
			dashesLen = 2
		}
		bottomBorder = GrayText(strings.Repeat("─", dashesLen)) + " " +
			GrayText(modelName) + " " + GrayText("·") + " " +
			"\033[1;38;5;218mmncode\033[0m \033[1;38;5;212mPRO\033[0m \033[1;38;5;219mMAX\033[0m \033[38;5;218m─\033[0m"
	} else {
		titleText := modelName
		if branch != "" {
			titleText = fmt.Sprintf("%s · %s", modelName, branch)
		}
		dashesLen := width - len([]rune(titleText)) - 4
		if dashesLen < 2 {
			dashesLen = 2
		}
		bottomBorder = GrayText(strings.Repeat("─", dashesLen) + " " + titleText + " ─")
	}

	var footer string
	proBadge := ""
	if isProMax {
		proBadge = "\033[1;38;5;218m[PRO MAX]\033[0m "
	}

	agentHint := "← for agents"
	if s.Subagents != nil && len(s.Subagents.List()) > 0 {
		agentHint = fmt.Sprintf("← for agents (%d)", len(s.Subagents.List()))
	}

	copyToast := GetActiveCopyToast()
	updateNotice := GetCachedUpdateNotice()
	idleRizz := GetActiveIdleRizz()
	if copyToast != "" {
		footer = fmt.Sprintf("  %s%s%s", proBadge, copyToast,
			GrayText(fmt.Sprintf(" · (shift+tab to cycle) · %s", agentHint)))
	} else if updateNotice != "" {
		footer = fmt.Sprintf("  %s%s", proBadge, updateNotice)
	} else if idleRizz != "" {
		footer = fmt.Sprintf("  %s\033[1;38;5;218m[RIZZ IDLE]\033[0m \033[38;5;225m%s\033[0m", proBadge, idleRizz)
	} else {
		switch s.Config.PermissionMode {
		case config.PermissionModeBypass:
			footer = fmt.Sprintf("  %s%s%s", proBadge, BoldYellow("⏵⏵ bypass permissions on"),
				GrayText(fmt.Sprintf(" (shift+tab to cycle) · %s · drag to auto-copy", agentHint)))
		case config.PermissionModeAuto:
			footer = fmt.Sprintf("  %s%s%s", proBadge, BoldCyan("⏵ auto-approve on"),
				GrayText(fmt.Sprintf(" (shift+tab to cycle) · %s · drag to auto-copy", agentHint)))
		default:
			footer = fmt.Sprintf("  %s%s", proBadge,
				GrayText(fmt.Sprintf("ask permissions (shift+tab to cycle) · %s · drag to auto-copy", agentHint)))
		}
	}

	topBorder := GrayText(strings.Repeat("─", width))

	var lines []string
	lines = append(lines, topBorder, fmt.Sprintf("%s %s", promptPrefix, highlightPromptInput(string(input))))

	items, _, _ := GetActiveDropdownItems(s, input, cursorPos)
	dropdownLines, dropdownCount := RenderDropdownLines(items, selectedIdx, width)
	if len(dropdownLines) > 0 {
		lines = append(lines, dropdownLines...)
	}

	lines = append(lines, bottomBorder, footer)
	return lines, dropdownCount, promptPrefix, promptLen
}

func truncateText(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > maxLen {
		if maxLen <= 3 {
			return string(runes[:maxLen])
		}
		return string(runes[:maxLen-3]) + "…"
	}
	return s
}

func highlightPromptInput(input string) string {
	if strings.HasPrefix(input, "/") {
		parts := strings.SplitN(input, " ", 2)
		cmd := parts[0]
		highlightedCmd := BoldCyan(cmd)
		if len(parts) > 1 {
			return highlightedCmd + " " + highlightAtMentions(parts[1])
		}
		return highlightedCmd
	}
	return highlightAtMentions(input)
}

func highlightAtMentions(text string) string {
	return atMentionRegex.ReplaceAllStringFunc(text, func(m string) string {
		return "\033[1;38;5;218m" + m + "\033[0m"
	})
}
