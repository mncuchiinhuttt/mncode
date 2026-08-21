package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"strings"
	"time"
)

func buildPromptLines(s *agent.Session, input []rune, cursorPos int, selectedIdx int, width int, isProMax bool, branch string) ([]string, int, int, string, int) {
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
	activeCount := 0
	if s.Subagents != nil {
		activeCount = s.Subagents.ActiveCount()
		if activeCount == 1 {
			agentHint = "← 1 agent"
		} else if activeCount > 1 {
			agentHint = fmt.Sprintf("← %d agents", activeCount)
		} else if len(s.Subagents.List()) > 0 {
			agentHint = fmt.Sprintf("← for agents (%d)", len(s.Subagents.List()))
		}
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
		isBrainrot := s.Config.GetSetting("brainrot_mode", "false") == "true"
		switch s.Config.PermissionMode {
		case config.PermissionModePlan:
			label := BoldPastelPink("⏵⏵ plan mode on (read-only)")
			if isBrainrot {
				label = BoldPastelPink("⏵⏵ plan mode (200 IQ overthinking, zero code touched fr fr)")
			}
			footer = fmt.Sprintf("  %s%s%s", proBadge, label,
				GrayText(fmt.Sprintf(" · (shift+tab to cycle) · %s · drag to auto-copy", agentHint)))
		case config.PermissionModeBypass:
			label := BoldYellow("⏵⏵ bypass permissions on")
			if isBrainrot {
				label = BoldYellow("⏵⏵ full bypass (high risk high reward, let him cook with max rizz)")
			}
			footer = fmt.Sprintf("  %s%s%s", proBadge, label,
				GrayText(fmt.Sprintf(" · (shift+tab to cycle) · %s · drag to auto-copy", agentHint)))
		case config.PermissionModeAuto:
			label := BoldCyan("⏵ auto-approve on")
			if isBrainrot {
				label = BoldCyan("⏵ auto-cooking on (speedrunning tech debt, zero cap)")
			}
			footer = fmt.Sprintf("  %s%s%s", proBadge, label,
				GrayText(fmt.Sprintf(" · (shift+tab to cycle) · %s · drag to auto-copy", agentHint)))
		default:
			label := GrayText("ask permissions")
			if isBrainrot {
				label = GrayText("asking permissions (low risk low aura, seeking adult supervision)")
			}
			footer = fmt.Sprintf("  %s%s%s", proBadge, label,
				GrayText(fmt.Sprintf(" · (shift+tab to cycle) · %s · drag to auto-copy", agentHint)))
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

	// Claude-styled live subagents list underneath the footer
	subagentCount := 0
	if s.Subagents != nil && len(s.Subagents.List()) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s %s", BoldCyan("●"), Bold("main")))
		subagentCount += 2

		for _, sub := range s.Subagents.List() {
			circle := BoldCyan("◯")
			if sub.Status == "completed" {
				circle = BoldGreen("✓")
			} else if sub.Status == "error" {
				circle = BoldRed("✗")
			}

			elapsed := time.Since(sub.StartTime)
			if !sub.EndTime.IsZero() {
				elapsed = sub.Duration
			}
			mins := int(elapsed.Minutes())
			secs := int(elapsed.Seconds()) % 60
			timeStr := fmt.Sprintf("%ds", secs)
			if mins > 0 {
				timeStr = fmt.Sprintf("%dm %ds", mins, secs)
			}

			tokensStr := ""
			if sub.Tokens > 0 {
				tokensStr = fmt.Sprintf(" · ↓ %s tokens", formatTokens(sub.Tokens))
			} else if len(sub.ToolCalls) > 0 {
				tokensStr = fmt.Sprintf(" · %d tool calls", len(sub.ToolCalls))
			}

			metaStr := fmt.Sprintf("%s%s", GrayText(timeStr), GrayText(tokensStr))
			activity := sub.CurrentActivity
			if activity == "" {
				activity = sub.Prompt
			}
			if len([]rune(activity)) > 42 {
				activity = string([]rune(activity)[:41]) + "…"
			}

			gap := width - len(sub.Name) - len([]rune(activity)) - len([]rune(timeStr+tokensStr)) - 10
			if gap < 2 {
				gap = 2
			}

			agentLine := fmt.Sprintf("  %s %s  %s%s%s",
				circle, Bold(sub.Name), GrayText(activity), strings.Repeat(" ", gap), metaStr)
			lines = append(lines, agentLine)
			subagentCount++
		}
	}

	return lines, dropdownCount, subagentCount, promptPrefix, promptLen
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
	res := atMentionRegex.ReplaceAllStringFunc(text, func(m string) string {
		return "\033[1;38;5;218m" + m + "\033[0m"
	})
	res = FormatPastedTagHighlight(res)
	if strings.Contains(res, "[Image:") {
		res = strings.ReplaceAll(res, "[Image:", "\033[1;38;5;213m[Image:\033[0;38;5;225m")
	}
	return res
}
