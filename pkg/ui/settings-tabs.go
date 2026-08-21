package ui

import (
	"encoding/json"
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"runtime"
	"strings"
)

func RenderSettingsTab(s *agent.Session, allDefs []SettingDef, searchQuery string, selectedIdx, scrollOffset, width int) ([]string, int, int) {
	var lines []string
	searchContent := searchQuery
	if searchContent == "" {
		searchContent = GrayText("Search settings…")
	}
	boxWidth := width - 6
	if boxWidth < 35 {
		boxWidth = 35
	}
	maxSearchLen := boxWidth - 4
	cleanSearch := stripAnsi(searchContent)
	runes := []rune(cleanSearch)
	if len(runes) > maxSearchLen {
		searchContent = string(runes[:maxSearchLen])
		cleanSearch = string(runes[:maxSearchLen])
	}

	padLen := boxWidth - 4 - len([]rune(cleanSearch))
	if padLen < 0 {
		padLen = 0
	}

	topBox := "  ╭" + strings.Repeat("─", boxWidth) + "╮"
	midBox := fmt.Sprintf("  │ ⌕ %s%s │", searchContent, strings.Repeat(" ", padLen))
	botBox := "  ╰" + strings.Repeat("─", boxWidth) + "╯"
	lines = append(lines, topBox, midBox, botBox, "")

	var filtered []SettingDef
	if strings.TrimSpace(searchQuery) == "" {
		filtered = allDefs
	} else {
		q := strings.ToLower(searchQuery)
		for _, d := range allDefs {
			if strings.Contains(strings.ToLower(d.Label), q) || strings.Contains(strings.ToLower(d.Key), q) {
				filtered = append(filtered, d)
			}
		}
	}

	if len(filtered) == 0 {
		lines = append(lines, GrayText("     No matching settings found."))
		return lines, selectedIdx, scrollOffset
	}

	if selectedIdx >= len(filtered) {
		selectedIdx = len(filtered) - 1
	}
	if selectedIdx < 0 {
		selectedIdx = 0
	}
	maxDisplay := 10
	if selectedIdx < scrollOffset {
		scrollOffset = selectedIdx
	} else if selectedIdx >= scrollOffset+maxDisplay {
		scrollOffset = selectedIdx - maxDisplay + 1
	}

	endIdx := scrollOffset + maxDisplay
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	if scrollOffset > 0 {
		lines = append(lines, GrayText(fmt.Sprintf("    ↑ %d more settings above", scrollOffset)))
	}

	for i := scrollOffset; i < endIdx; i++ {
		def := filtered[i]
		val := GetSettingValue(s, def)
		prefix := "    "
		labelStr := def.Label
		if i == selectedIdx {
			prefix = BoldPastelPink("  ❯ ")
			labelStr = Bold(def.Label)
		}

		valStr := val
		if val == "true" {
			valStr = BoldGreen("true")
		} else if val == "false" {
			valStr = GrayText("false")
		} else {
			valStr = BoldCyan(val)
		}

		colWidth := boxWidth - 22
		if colWidth < 35 {
			colWidth = 35
		}
		lines = append(lines, fmt.Sprintf("%s%-*s %s", prefix, colWidth, labelStr, valStr))
	}

	if endIdx < len(filtered) {
		remainingBelow := len(filtered) - endIdx
		lines = append(lines, GrayText(fmt.Sprintf("    ↓ %d more settings below", remainingBelow)))
	}

	return lines, selectedIdx, scrollOffset
}

func RenderStatusTab(s *agent.Session) []string {
	var lines []string
	lines = append(lines, BoldCyan("  System & Environment Status:"))
	lines = append(lines, fmt.Sprintf("    • %-20s : %s/%s", "Platform / Arch", runtime.GOOS, runtime.GOARCH))
	lines = append(lines, fmt.Sprintf("    • %-20s : %s", "Go Version", runtime.Version()))
	lines = append(lines, fmt.Sprintf("    • %-20s : %s", "Workspace", s.WorkspaceDir))
	lines = append(lines, fmt.Sprintf("    • %-20s : %s", "Git Branch", GetGitBranchOrFolder(s.WorkspaceDir)))
	if s.Catalog != nil {
		lines = append(lines, fmt.Sprintf("    • %-20s : %d skills, %d agents, %d rules",
			"Loaded Catalog", len(s.Catalog.Skills), len(s.Catalog.Agents), len(s.Catalog.Rules)))
	}
	if s.Accounts != nil {
		lines = append(lines, fmt.Sprintf("    • %-20s : %d accounts registered", "Account Pool", len(s.Accounts.Accounts)))
	}
	return lines
}

func RenderConfigTab(s *agent.Session) []string {
	var lines []string
	path, _ := config.GetConfigFilePath()
	lines = append(lines, fmt.Sprintf("  %s %s", BoldCyan("Raw Config File:"), GrayText(path)))
	lines = append(lines, "")

	data, err := json.MarshalIndent(s.Config, "    ", "  ")
	if err == nil {
		jsonLines := strings.Split(string(data), "\n")
		maxL := 14
		if len(jsonLines) < maxL {
			maxL = len(jsonLines)
		}
		for i := 0; i < maxL; i++ {
			lines = append(lines, "    "+jsonLines[i])
		}
		if len(jsonLines) > maxL {
			lines = append(lines, GrayText(fmt.Sprintf("    ... (+%d more lines)", len(jsonLines)-maxL)))
		}
	}
	return lines
}

func RenderUsageTab(s *agent.Session) []string {
	var lines []string
	lines = append(lines, BoldCyan("  Live Quota & Usage Overview:"))
	if s.Accounts != nil && len(s.Accounts.Accounts) > 0 {
		for _, acc := range s.Accounts.Accounts {
			status := BoldGreen("[ACTIVE]")
			if !acc.IsActive {
				status = GrayText("[STANDBY]")
			}
			lines = append(lines, fmt.Sprintf("    • %-32s %s (%s)", Bold(acc.Email), status, acc.Provider))
		}
	} else {
		lines = append(lines, GrayText("    No accounts in pool. Run '/login antigravity' to connect."))
	}
	lines = append(lines, "")
	lines = append(lines, GrayText("  Type '/quota' in main prompt for full visual token breakdown & reset times."))
	return lines
}

func RenderStatsTab(s *agent.Session) []string {
	var lines []string
	lines = append(lines, BoldCyan("  Session Statistics:"))
	lines = append(lines, fmt.Sprintf("    • %-20s : %s", "Active Model", BoldGreen(s.Config.Model)))
	lines = append(lines, fmt.Sprintf("    • %-20s : %s", "Provider", BoldCyan(string(s.Config.Provider))))
	lines = append(lines, fmt.Sprintf("    • %-20s : %d", "Messages in Session", len(s.History)))
	if s.Subagents != nil {
		lines = append(lines, fmt.Sprintf("    • %-20s : %d subagents spawned", "Subagents Run", len(s.Subagents.List())))
	}
	return lines
}
