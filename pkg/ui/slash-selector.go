package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"strings"

	"github.com/manifoldco/promptui"
)

type SlashOption struct {
	Command     string
	Description string
	Category    string
}

var slashOptions = []SlashOption{
	{Command: "/agents", Description: "List all mnCode specialized agents (planner, tester...)", Category: "Subagents"},
	{Command: "/account", Description: "Manage accounts: switch primary, logout, remove, add", Category: "Accounts"},
	{Command: "/account list", Description: "View multi-account pool status", Category: "Accounts"},
	{Command: "/account import", Description: "Auto-import Antigravity & OpenAI credentials", Category: "Accounts"},
	{Command: "/login", Description: "Interactive login to Antigravity (Gemini) or Codex (OpenAI)", Category: "Accounts"},
	{Command: "/model", Description: "View or change current LLM model", Category: "Settings"},
	{Command: "/effort", Description: "Configure thinking budget / reasoning effort (low/med/high/max)", Category: "Settings"},
	{Command: "/goal", Description: "Run persistent autonomous goal loop with live query stopwatch", Category: "Goal"},
	{Command: "/workflow", Description: "Configure agent workflow mode (auto/ultracode/plan-first)", Category: "Workflow"},
	{Command: "/theme", Description: "Browse & switch color theme (Pastel Pink, Dark, Light...)", Category: "Theme"},
	{Command: "/skills", Description: "Browse, search, and activate ClaudeKit & workspace skills", Category: "Skills"},
	{Command: "/context", Description: "Show context window usage and breakdown bar", Category: "Context"},
	{Command: "/compact", Description: "Compress conversation history to free context", Category: "Context"},
	{Command: "/quota", Description: "Check account health, quota, and model availability", Category: "Accounts"},
	{Command: "/usage", Description: "View daily, monthly, and lifetime token usage stats", Category: "Stats"},
	{Command: "/config", Description: "View or update settings (language, model, themes, permissions)", Category: "Settings"},
	{Command: "/btw", Description: "Ask a quick side question without polluting task history", Category: "General"},
	{Command: "/brainrot", Description: "Toggle Gen Z / Sigma developer personality mode on/off", Category: "Settings"},
	{Command: "/resume", Description: "Browse & resume previous conversation sessions", Category: "Session"},
	{Command: "/history", Description: "View conversation history and past turns", Category: "Session"},
	{Command: "/status", Description: "Show session configuration & message stats", Category: "Settings"},
	{Command: "/scan", Description: "Deep scan & prime entire codebase architecture into memory", Category: "General"},
	{Command: "/research", Description: "Autonomous Deep Research pipeline with web search & full reports", Category: "Research"},
	{Command: "/litrev", Description: "Academic Literature Review pipeline with taxonomy & comparisons", Category: "Research"},
	{Command: "/mcp", Description: "Manage and connect Model Context Protocol (MCP) servers", Category: "Tools"},
	{Command: "/recap", Description: "Synthesize structured session recap & files touched", Category: "Session"},
	{Command: "/update", Description: "Check GitHub Releases and self-update mncode to latest version", Category: "General"},
	{Command: "/version", Description: "Show current mncode version and platform architecture", Category: "Help"},
	{Command: "/help", Description: "Show help and command palette", Category: "Help"},
	{Command: "/exit", Description: "Exit mncode assistant", Category: "Session"},
}

// OpenInteractiveSlashMenu opens an arrow-navigable, searchable menu for slash commands
func OpenInteractiveSlashMenu(s *agent.Session) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "\033[1;36m> {{ .Command | cyan | bold }} \033[0m - {{ .Description | faint }} ({{ .Category | yellow }})",
		Inactive: "  {{ .Command | cyan }} - {{ .Description | faint }}",
		Selected: "\033[1;32m[OK] {{ .Command | bold }}\033[0m",
		Details: `
--------- Command Details ---------
{{ "Command:" | bold }}     {{ .Command | cyan }}
{{ "Category:" | bold }}    {{ .Category | yellow }}
{{ "Description:" | bold }} {{ .Description }}`,
	}

	searcher := func(input string, index int) bool {
		opt := slashOptions[index]
		q := strings.ToLower(strings.TrimPrefix(input, "/"))
		name := strings.ToLower(strings.TrimPrefix(opt.Command, "/"))
		desc := strings.ToLower(opt.Description)
		cat := strings.ToLower(opt.Category)

		return strings.Contains(name, q) || strings.Contains(desc, q) || strings.Contains(cat, q)
	}

	prompt := promptui.Select{
		Label:        BoldCyan("Select a Slash Command (Use arrows or type to filter)"),
		Items:        slashOptions,
		Templates:    templates,
		Size:         8,
		Searcher:     searcher,
		HideSelected: true,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return
	}

	chosen := slashOptions[idx]
	fmt.Printf("\nExecuting %s...\n", BoldCyan(chosen.Command))
	HandleSlashCommand(chosen.Command, s)
}
