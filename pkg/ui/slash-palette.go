package ui

import (
	"fmt"
	"strconv"
	"strings"
)

type SlashMenuItem struct {
	Number      int
	Command     string
	Description string
}

var slashMenuItems = []SlashMenuItem{
	{Number: 1, Command: "/skills", Description: "List all loaded Claude & mnCode skills"},
	{Number: 2, Command: "/agents", Description: "List all mnCode specialized agents (planner, tester...)"},
	{Number: 3, Command: "/rules", Description: "View active workspace development rules"},
	{Number: 4, Command: "/account", Description: "Manage accounts: switch primary, logout, remove, add"},
	{Number: 5, Command: "/account import", Description: "Auto-import Antigravity & OpenAI credentials"},
	{Number: 6, Command: "/login antigravity", Description: "Interactive login with Antigravity token"},
	{Number: 7, Command: "/login codex", Description: "Interactive login with Codex/OpenAI API key"},
	{Number: 8, Command: "/model", Description: "Browse & switch AI models (Gemini 2.5 Pro, Claude 3.7 Sonnet...)"},
	{Number: 9, Command: "/effort", Description: "Configure thinking budget / reasoning effort (low/med/high/max)"},
	{Number: 10, Command: "/goal", Description: "Run persistent autonomous goal loop with live query stopwatch"},
	{Number: 11, Command: "/workflow", Description: "Configure agent workflow mode (auto/ultra-workflow/plan-first)"},
	{Number: 12, Command: "/theme", Description: "Browse & switch color theme (Pastel Pink, Dark, Light, Cyberpunk...)"},
	{Number: 13, Command: "/skills", Description: "Browse, search, and activate ClaudeKit & workspace skills"},
	{Number: 14, Command: "/context", Description: "Show context window usage and breakdown bar"},
	{Number: 15, Command: "/compact", Description: "Compress conversation history to free context"},
	{Number: 16, Command: "/quota", Description: "Check account health, quota, and model availability"},
	{Number: 17, Command: "/usage", Description: "View daily, monthly, and lifetime token usage stats"},
	{Number: 18, Command: "/config", Description: "View or update settings and ~/.mncode/config.json"},
	{Number: 19, Command: "/status", Description: "Show session configuration & message stats"},
	{Number: 20, Command: "/clear", Description: "Clear terminal screen and conversation history"},
	{Number: 21, Command: "/help", Description: "Show available slash commands"},
	{Number: 22, Command: "/exit", Description: "Exit mncode assistant"},
}

// ShowSlashPalette prints a clean list of available slash commands without boxes or emojis
func ShowSlashPalette() {
	fmt.Println()
	fmt.Println(BoldCyan("AVAILABLE SLASH COMMANDS:"))
	for _, item := range slashMenuItems {
		fmt.Printf("  %-4s %-20s %s\n",
			BoldYellow(fmt.Sprintf("[%d]", item.Number)),
			BoldCyan(item.Command),
			GrayText(item.Description))
	}
	fmt.Println()
	fmt.Println(GrayText("  Tip: Type command name (e.g. /context, /compact) or shortcut number (e.g. /12, /13)"))
	fmt.Println()
}

// ResolveNumericSlashCommand checks if input is like "/1" or "/4" and returns full command
func ResolveNumericSlashCommand(input string) string {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return input
	}
	numStr := strings.TrimPrefix(trimmed, "/")
	if num, err := strconv.Atoi(numStr); err == nil {
		for _, item := range slashMenuItems {
			if item.Number == num {
				return item.Command
			}
		}
	}
	return input
}
