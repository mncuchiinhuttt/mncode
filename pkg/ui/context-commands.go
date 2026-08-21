package ui

import (
	"context"
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"

	"golang.org/x/term"
)

// HandleContextCommand handles /context slash command variations
func HandleContextCommand(parts []string, s *agent.Session) {
	if len(parts) == 1 {
		ShowContextUsage(s)
		return
	}

	arg := strings.ToLower(parts[1])
	switch arg {
	case "window", "size", "slider":
		if len(parts) > 2 {
			setContextWindowDirect(s, parts[2])
		} else {
			OpenInteractiveContextWindowSlider(s)
		}
	case "200k", "300k", "500k", "1m":
		setContextWindowDirect(s, arg)
	case "compact", "compress":
		HandleCompactCommand(s)
	default:
		ShowContextUsage(s)
	}
}

func setContextWindowDirect(s *agent.Session, val string) {
	lower := strings.ToLower(val)
	s.Config.ContextWindow = lower
	s.Config.SetSetting("context_window", lower)
	_ = config.SaveConfig(s.Config)
	fmt.Printf("\n%s Context Window Size set to: %s (%d tokens)\n\n",
		BoldGreen("[Context]"), BoldCyan(s.Config.GetContextWindowLabel()), s.Config.GetContextWindowTokens())
}

// ShowContextUsage renders visual context window utilization as an interactive overlay
func ShowContextUsage(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		renderContextUsageStatic(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		renderContextUsageStatic(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	lastLinesCount := 0

	render := func() {
		usage := s.GetContextUsage()

		totalBlocks := 100
		usedBlocks := int((float64(usage.TotalUsed) / float64(usage.Limit)) * 100.0)
		if usedBlocks < 1 && usage.TotalUsed > 0 {
			usedBlocks = 1
		}
		if usedBlocks > totalBlocks {
			usedBlocks = totalBlocks
		}

		bufferBlocks := 3
		if bufferBlocks > totalBlocks-usedBlocks {
			bufferBlocks = totalBlocks - usedBlocks
		}

		sysBlocks := int((float64(usage.SystemTokens) / float64(usage.Limit)) * 100.0)
		toolsBlocks := int((float64(usage.SystemToolsTokens) / float64(usage.Limit)) * 100.0)
		mcpBlocks := int((float64(usage.MCPToolsTokens) / float64(usage.Limit)) * 100.0)
		skillsBlocks := int((float64(usage.SkillsTokens) / float64(usage.Limit)) * 100.0)

		symbols := make([]string, totalBlocks)
		for i := 0; i < totalBlocks; i++ {
			if i < sysBlocks {
				symbols[i] = BoldCyan("⛁")
			} else if i < sysBlocks+toolsBlocks {
				symbols[i] = BoldGreen("⛁")
			} else if i < sysBlocks+toolsBlocks+mcpBlocks {
				symbols[i] = BoldYellow("⛁")
			} else if i < sysBlocks+toolsBlocks+mcpBlocks+skillsBlocks {
				symbols[i] = BoldMagenta("⛁")
			} else if i < usedBlocks {
				symbols[i] = BoldBlue("⛁")
			} else if i >= totalBlocks-bufferBlocks {
				symbols[i] = BoldYellow("⛝")
			} else {
				symbols[i] = GrayText("⛶")
			}
		}

		rightLines := make([]string, 10)
		rightLines[0] = Bold(usage.DisplayName)
		rightLines[1] = GrayText(fmt.Sprintf("%s · %s Window", usage.Model, s.Config.GetContextWindowLabel()))
		rightLines[2] = fmt.Sprintf("%s/%s tokens (%.0f%%)",
			formatTokens(usage.TotalUsed), formatTokens(usage.Limit), usage.PercentUsed)
		rightLines[3] = ""
		rightLines[4] = Bold("Estimated usage by category")
		rightLines[5] = fmt.Sprintf("%s System prompt: %s tokens (%.1f%%)",
			BoldCyan("⛁"), formatTokens(usage.SystemTokens), pct(usage.SystemTokens, usage.Limit))
		rightLines[6] = fmt.Sprintf("%s System tools: %s tokens (%.1f%%)",
			BoldGreen("⛁"), formatTokens(usage.SystemToolsTokens), pct(usage.SystemToolsTokens, usage.Limit))
		rightLines[7] = fmt.Sprintf("%s MCP tools: %s tokens (%.1f%%)",
			BoldYellow("⛁"), formatTokens(usage.MCPToolsTokens), pct(usage.MCPToolsTokens, usage.Limit))
		rightLines[8] = fmt.Sprintf("%s Skills: %s tokens (%.1f%%)",
			BoldMagenta("⛁"), formatTokens(usage.SkillsTokens), pct(usage.SkillsTokens, usage.Limit))
		rightLines[9] = fmt.Sprintf("%s Messages: %s tokens (%.1f%%)",
			BoldBlue("⛁"), formatTokens(usage.MessagesTokens), pct(usage.MessagesTokens, usage.Limit))

		var lines []string
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s  %s  %s", BoldCyan("⎿"), Bold("Context Window Usage"), GrayText("(Esc/q exit · w window size · c compact)")))
		lines = append(lines, "")

		for r := 0; r < 10; r++ {
			var rowSymbols []string
			for c := 0; c < 10; c++ {
				rowSymbols = append(rowSymbols, symbols[r*10+c])
			}
			gridStr := strings.Join(rowSymbols, " ")
			lines = append(lines, fmt.Sprintf("     %s   %s", gridStr, rightLines[r]))
		}

		freePct := pct(usage.RemainingTokens, usage.Limit)
		bufPct := pct(usage.AutoCompactBuffer, usage.Limit)

		lines = append(lines, fmt.Sprintf("                           %s Free space: %s (%.1f%%)",
			GrayText("⛶"), formatTokens(usage.RemainingTokens), freePct))
		lines = append(lines, fmt.Sprintf("                           %s Autocompact buffer: %s tokens (%.1f%%)",
			BoldYellow("⛝"), formatTokens(usage.AutoCompactBuffer), bufPct))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("     Window Size: %s tokens · %d tools · %d skills · %s",
			BoldCyan(formatTokens(usage.Limit)), usage.ToolCount, usage.SkillsCount, GrayText("Press 'w' to change size")))
		lines = append(lines, "")

		if lastLinesCount > 0 {
			fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
		}

		for i, line := range lines {
			if i < len(lines)-1 {
				fmt.Printf("\r\033[K%s\r\n", line)
			} else {
				fmt.Printf("\r\033[K%s", line)
			}
		}
		lastLinesCount = len(lines)
	}

	render()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		b := buf[0]
		switch b {
		case 3, 27, 'q', 'Q', 13, 10:
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return

		case 'w', 'W': // Change window size
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
			OpenInteractiveContextWindowSlider(s)
			return

		case 'c', 'C': // Compact history
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
			HandleCompactCommand(s)
			return
		}
	}
}

func renderContextUsageStatic(s *agent.Session) {
	usage := s.GetContextUsage()
	fmt.Printf("\nContext Usage: %s/%s tokens (%.1f%%) · Window: %s\n",
		formatTokens(usage.TotalUsed), formatTokens(usage.Limit), usage.PercentUsed, s.Config.GetContextWindowLabel())
}

// HandleCompactCommand triggers conversation history compression
func HandleCompactCommand(s *agent.Session) {
	if len(s.History) <= 2 {
		fmt.Println("Conversation history is too short to compact (requires at least 3 messages).")
		return
	}

	fmt.Println("Compacting conversation history into structured summary checkpoint...")
	ctx := context.Background()

	result, err := s.CompactHistory(ctx)
	if err != nil {
		fmt.Printf("%s %v\n", BoldRed("[Error] Compaction failed:"), err)
		return
	}

	fmt.Println()
	fmt.Println(BoldGreen("[Success] Context Compaction Complete:"))
	fmt.Printf("  • Previous Usage: %s tokens\n", Bold(formatTokens(result.OriginalTokens)))
	fmt.Printf("  • Compact Usage:  %s tokens\n", BoldGreen(formatTokens(result.CompactTokens)))
	fmt.Printf("  • Space Freed:    %s tokens (%s reduction)\n",
		BoldCyan(formatTokens(result.FreedTokens)),
		BoldYellow(fmt.Sprintf("%.1f%%", result.PercentFreed)))
	if result.SnapshotFile != "" {
		fmt.Printf("  • Backup Saved:   %s\n", GrayText(result.SnapshotFile))
	}
	fmt.Println()
}

func formatTokens(tokens int) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000.0)
	}
	if tokens >= 1000 {
		return fmt.Sprintf("%.1fk", float64(tokens)/1000.0)
	}
	return fmt.Sprintf("%d", tokens)
}

func pct(part, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return (float64(part) / float64(total)) * 100.0
}
