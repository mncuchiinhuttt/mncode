package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"strings"
)

func buildPromptLines(s *agent.Session, input []rune, selectedIdx int, width int, isProMax bool, branch string) ([]string, int, string, int) {
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
	if copyToast != "" {
		footer = fmt.Sprintf("  %s%s%s", proBadge, copyToast,
			GrayText(fmt.Sprintf(" · (shift+tab to cycle) · %s", agentHint)))
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

	matching := getMatchingSlashOptions(s, string(input))
	dropdownCount := 0
	if len(matching) > 0 && strings.HasPrefix(string(input), "/") {
		maxDisplay := 5
		if len(matching) < maxDisplay {
			maxDisplay = len(matching)
		}
		startIdx := 0
		if selectedIdx >= maxDisplay {
			startIdx = selectedIdx - maxDisplay + 1
		}
		endIdx := startIdx + maxDisplay
		if endIdx > len(matching) {
			endIdx = len(matching)
			startIdx = endIdx - maxDisplay
			if startIdx < 0 {
				startIdx = 0
			}
		}
		cmdColWidth := 26
		for i := startIdx; i < endIdx; i++ {
			opt := matching[i]
			cmdStr := opt.Command
			cmdRunes := []rune(cmdStr)
			if len(cmdRunes) > cmdColWidth {
				cmdStr = string(cmdRunes[:cmdColWidth-1]) + "…"
			} else {
				cmdStr = cmdStr + strings.Repeat(" ", cmdColWidth-len(cmdRunes))
			}

			maxDescLen := width - 4 - cmdColWidth - 4
			if maxDescLen < 0 {
				maxDescLen = 0
			}
			descStr := truncateText(opt.Description, maxDescLen)

			if i == selectedIdx {
				lines = append(lines, fmt.Sprintf("  \033[1;36m> %s\033[0m \033[90m%s\033[0m", cmdStr, descStr))
			} else {
				lines = append(lines, fmt.Sprintf("    \033[36m%s\033[0m \033[2m%s\033[0m", cmdStr, descStr))
			}
		}
		dropdownCount = endIdx - startIdx
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
	if !strings.HasPrefix(input, "/") {
		return input
	}
	parts := strings.SplitN(input, " ", 2)
	cmd := parts[0]
	if len(parts) == 1 {
		return BoldCyan(cmd)
	}
	return BoldCyan(cmd) + " " + parts[1]
}
