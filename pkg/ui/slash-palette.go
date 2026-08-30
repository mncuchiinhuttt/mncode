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
	{Number: 6, Command: "/account login antigravity", Description: "Interactive login with Antigravity token"},
	{Number: 7, Command: "/account login codex", Description: "Interactive login with Codex/OpenAI API key"},
	{Number: 8, Command: "/model", Description: "Browse & switch AI models (Gemini 2.5 Pro, Claude 3.7 Sonnet...)"},
	{Number: 9, Command: "/effort", Description: "Configure thinking budget / reasoning effort (low/med/high/max)"},
	{Number: 10, Command: "/goal", Description: "Run persistent autonomous goal loop with live query stopwatch"},
	{Number: 11, Command: "/workflow", Description: "Configure agent workflow mode (auto/ultra-workflow/plan-first)"},
	{Number: 12, Command: "/theme", Description: "Browse & switch color theme (Pastel Pink, Dark, Light, Cyberpunk...)"},
	{Number: 13, Command: "/context", Description: "Show context window usage and breakdown bar"},
	{Number: 14, Command: "/compact", Description: "Compress conversation history to free context"},
	{Number: 15, Command: "/quota", Description: "Check account health, quota, and model availability"},
	{Number: 16, Command: "/usage", Description: "View daily, monthly, and lifetime token usage stats"},
	{Number: 17, Command: "/config", Description: "View or update settings and ~/.mncode/config.json"},
	{Number: 18, Command: "/status", Description: "Show session configuration & message stats"},
	{Number: 19, Command: "/diff", Description: "View uncommitted git diffs and modified files in workspace"},
	{Number: 20, Command: "/undo", Description: "Revert last agent turn and restore workspace files"},
	{Number: 21, Command: "/rewind", Description: "Rewind N previous turns and conversation steps"},
	{Number: 22, Command: "/checkpoint", Description: "Save or list manual and turn snapshots"},
	{Number: 23, Command: "/commit", Description: "AI-assisted semantic Conventional Commit & push"},
	{Number: 24, Command: "/test", Description: "Run project test suite automatically"},
	{Number: 25, Command: "/heal", Description: "Autonomous test failure analysis and self-healing loop"},
	{Number: 26, Command: "/review", Description: "Pre-commit security and clean code auditor"},
	{Number: 27, Command: "/share", Description: "Export and share session transcript to web & clipboard"},
	{Number: 28, Command: "/doctor", Description: "Diagnose workspace health, toolchains, and rule limits"},
	{Number: 29, Command: "/pr", Description: "Create GitHub Pull Request with automated AI summary"},
	{Number: 30, Command: "/symbol", Description: "Search codebase functions, structs, classes via AST"},
	{Number: 31, Command: "/scratch", Description: "Open local code sandbox to evaluate test snippets"},
	{Number: 32, Command: "/resolve", Description: "Autonomous Git merge conflict detector & resolver"},
	{Number: 33, Command: "/db", Description: "Database schema explorer and SQL query runner"},
	{Number: 34, Command: "/api", Description: "HTTP & REST API endpoint tester with latency meter"},
	{Number: 35, Command: "/tree", Description: "Interactive ASCII codebase file tree visualizer"},
	{Number: 36, Command: "/changelog", Description: "Synthesize release notes and update CHANGELOG.md"},
	{Number: 37, Command: "/steer", Description: "Inject real-time high-priority steering into agent thought loop"},
	{Number: 38, Command: "/queue", Description: "Enqueue prompts to execute automatically after current turn"},
	{Number: 39, Command: "/troll", Description: "Toggle harmless fake scare prank commands on/off"},
	{Number: 40, Command: "/feedback", Description: "Send feedback about mncode to the team"},
	{Number: 41, Command: "/clear", Description: "Clear terminal screen and conversation history"},
	{Number: 42, Command: "/remote", Description: "Launch web & mobile remote companion bridge with QR code"},
	{Number: 43, Command: "/help", Description: "Show available slash commands"},
	{Number: 44, Command: "/exit", Description: "Exit mncode assistant"},
	{Number: 45, Command: "/search", Description: "Configure web search (Auto, Brave, Tavily, Google)"},
	{Number: 46, Command: "/export", Description: "Export frozen session trajectory as ShareGPT JSON"},
	{Number: 47, Command: "/combo", Description: "Manage and run multi-agent Combos (Swarm & Pipelines)"},
	{Number: 48, Command: "/debate", Description: "Launch AI Agent Debate Arena between competing models"},
	{Number: 49, Command: "/service", Description: "Manage background supervised services (dev servers, watchers)"},
	{Number: 50, Command: "/budget", Description: "Set session token or dollar budget ceiling with soft/hard stops"},
	{Number: 51, Command: "/memory", Description: "View and manage shared workspace memories and self-reflections"},
	{Number: 52, Command: "/voice", Description: "Record speech from microphone and transcribe directly via Whisper"},
	{Number: 53, Command: "/drift", Description: "Detect architecture, dependency, and exported API drift against a baseline"},
	{Number: 54, Command: "/sandbox", Description: "Run bounded argv fixtures in a temporary copy without source mutation"},
	{Number: 55, Command: "/index", Description: "Build local BM25 plus AST code search index"},
	{Number: 56, Command: "/arena", Description: "Run independent red-team reviews against the current git diff"},
	{Number: 57, Command: "/replay", Description: "Record and inspect a scrubbed agent flight trace"},
	{Number: 58, Command: "/fork", Description: "Fork a replay trace prefix into a new active session"},
	{Number: 59, Command: "/spec", Description: "Create and check deterministic feature contracts and test matrices"},
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
