package ui

import (
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"strings"
)

type SettingType int

const (
	SettingTypeBool SettingType = iota
	SettingTypeChoice
	SettingTypeModel
)

type SettingDef struct {
	Key         string
	Label       string
	Type        SettingType
	DefaultVal  string
	Choices     []string
	Description string
}

func GetAllSettingsDefinitions() []SettingDef {
	return []SettingDef{
		{Key: "model", Label: "Active AI Model", Type: SettingTypeModel, DefaultVal: "gemini-3.7-flash-high", Description: "Primary LLM model used for conversation and coding"},
		{Key: "effort", Label: "Thinking Effort Budget", Type: SettingTypeChoice, DefaultVal: "high", Choices: []string{"none", "low", "medium", "high", "max", "pro-max"}, Description: "Reasoning token budget for extended thinking models"},
		{Key: "workflow", Label: "Workflow Orchestration", Type: SettingTypeChoice, DefaultVal: "auto", Choices: []string{"auto", "ultra-workflow", "plan-first", "direct"}, Description: "Agent workflow strategy and autonomous chaining mode"},
		{Key: "context_window", Label: "Context Window Size", Type: SettingTypeChoice, DefaultVal: "200K", Choices: []string{"200K", "300K", "500K", "1M"}, Description: "Maximum context token threshold before compression"},
		{Key: "auto_compact", Label: "Auto-Compact Memory", Type: SettingTypeBool, DefaultVal: "true", Description: "Automatically summarize history when context exceeds 85%"},
		{Key: "token_saver_concise", Label: "Token Saver: Concise Responses", Type: SettingTypeBool, DefaultVal: "false", Description: "Inject a brevity directive — shorter, denser answers on every turn"},
		{Key: "token_saver_compress_output", Label: "Token Saver: Compress Command Output", Type: SettingTypeBool, DefaultVal: "false", Description: "Instruct the agent to filter and truncate long command output with head, tail, and grep"},
		{Key: "token_saver_targeted_edits", Label: "Token Saver: Targeted Edits", Type: SettingTypeBool, DefaultVal: "false", Description: "Prefer search-and-replace edits over full-file rewrites — far fewer output tokens"},
		{Key: "token_saver_rtk", Label: "Token Saver: RTK Shell Compression", Type: SettingTypeBool, DefaultVal: "false", Description: "Prefix dev commands with the rtk CLI for 60-90% smaller outputs (requires rtk installed)"},
		{Key: "token_saver_headroom", Label: "Token Saver: Headroom Proxy", Type: SettingTypeBool, DefaultVal: "false", Description: "Route requests through a local headroom proxy that compresses context (requires headroom installed)"},
		{Key: "permission_mode", Label: "Tool Permission Mode", Type: SettingTypeChoice, DefaultVal: "Manual (Ask)", Choices: []string{"Manual (Ask)", "Auto-Approve", "Bypass Permissions"}, Description: "Confirmation policy for command and file tool execution"},
		{Key: "theme", Label: "UI Color Theme", Type: SettingTypeChoice, DefaultVal: "Pastel Pink", Choices: []string{"Pastel Pink", "Dark mode", "Light mode", "Cyberpunk", "Monokai", "Tokyo Night"}, Description: "Terminal color scheme and syntax highlight palette"},
		{Key: "diff_style", Label: "Code Diff Highlight", Type: SettingTypeChoice, DefaultVal: "Full-line Background", Choices: []string{"Full-line Background", "Syntax Text Only"}, Description: "Visual style for file modification diff blocks"},
		{Key: "language", Label: "Language / Ngôn ngữ", Type: SettingTypeChoice, DefaultVal: "Default (English)", Choices: []string{"Default (English)", "Vietnamese", "Japanese", "Chinese", "Spanish", "French", "German"}, Description: "Natural language instructions for system prompt and responses"},
		{Key: "brainrot_mode", Label: "Brainrot Mode (Gen Z / Sigma)", Type: SettingTypeBool, DefaultVal: "false", Description: "Toggle Gen Z 10x developer persona and funny meme commentary"},
		{Key: "show_branch_name", Label: "Show Git Branch in Prompt", Type: SettingTypeBool, DefaultVal: "true", Description: "Display current git branch next to prompt input"},
		{Key: "show_tips", Label: "Show Productivity Tips", Type: SettingTypeBool, DefaultVal: "true", Description: "Display rotating developer tips in header and completion bars"},
		{Key: "copy_on_select", Label: "Copy on Mouse Select", Type: SettingTypeBool, DefaultVal: "true", Description: "Auto-copy selected terminal text to OS clipboard with toast notification"},
		{Key: "artifacts", Label: "Structured Artifacts", Type: SettingTypeBool, DefaultVal: "true", Description: "Prompt AI to generate structured markdown artifacts for plans and docs"},
		{Key: "auto_mode_plan", Label: "Auto-Execute Plan Phase", Type: SettingTypeBool, DefaultVal: "true", Description: "Automatically proceed to implementation after planning completes"},
		{Key: "worktree_base", Label: "Worktree Base Ref", Type: SettingTypeChoice, DefaultVal: "current", Choices: []string{"current", "main", "fresh"}, Description: "Git base reference used for subagent workspace branch isolation"},
		{Key: "troll_mode", Label: "Troll Mode (Harmless Pranks)", Type: SettingTypeBool, DefaultVal: "false", Description: "Display occasional funny fake scare commands before executing safe tools"},
		{Key: "interrupt_mode", Label: "Default Message Action", Type: SettingTypeChoice, DefaultVal: "queue", Choices: []string{"queue", "steer"}, Description: "Default behavior for messages: queue after turn (default) vs steer ongoing thought"},
		{Key: "verbose_output", Label: "Verbose Debug Logging", Type: SettingTypeBool, DefaultVal: "false", Description: "Print raw LLM API payloads and tool execution debug traces"},
	}
}

func GetSettingValue(s *agent.Session, def SettingDef) string {
	switch def.Key {
	case "model":
		if s.Config.Model != "" {
			return s.Config.Model
		}
		return def.DefaultVal
	case "effort":
		if s.Config.Effort != "" {
			return s.Config.Effort
		}
		return def.DefaultVal
	case "workflow":
		if s.Config.Workflow != "" {
			return s.Config.Workflow
		}
		return def.DefaultVal
	case "permission_mode":
		if s.Config.PermissionMode == config.PermissionModeBypass {
			return "Bypass Permissions"
		} else if s.Config.AutoApprove || s.Config.PermissionMode == config.PermissionModeAuto {
			return "Auto-Approve"
		}
		return "Manual (Ask)"
	case "context_window":
		return s.Config.GetContextWindowLabel()
	default:
		return s.Config.GetSetting(def.Key, def.DefaultVal)
	}
}

func ToggleOrCycleSetting(s *agent.Session, def SettingDef) {
	if def.Type == SettingTypeBool {
		cur := s.Config.GetSetting(def.Key, def.DefaultVal)
		newVal := "true"
		if cur == "true" {
			newVal = "false"
		}
		s.Config.SetSetting(def.Key, newVal)

		if def.Key == "verbose_output" {
			s.Config.Verbose = (newVal == "true")
		}
		_ = config.SaveConfig(s.Config)
		return
	}

	if def.Type == SettingTypeChoice && len(def.Choices) > 0 {
		cur := GetSettingValue(s, def)
		nextIdx := 0
		for i, c := range def.Choices {
			if strings.EqualFold(c, cur) {
				nextIdx = (i + 1) % len(def.Choices)
				break
			}
		}
		chosen := def.Choices[nextIdx]
		s.Config.SetSetting(def.Key, chosen)

		switch def.Key {
		case "effort":
			s.Config.Effort = chosen
		case "workflow":
			s.Config.Workflow = chosen
		case "permission_mode":
			switch chosen {
			case "Auto-Approve":
				s.Config.PermissionMode = config.PermissionModeAuto
				s.Config.AutoApprove = true
			case "Bypass Permissions":
				s.Config.PermissionMode = config.PermissionModeBypass
				s.Config.AutoApprove = true
			default:
				s.Config.PermissionMode = config.PermissionModeAsk
				s.Config.AutoApprove = false
			}
		case "context_window":
			s.Config.ContextWindow = chosen
		}

		_ = config.SaveConfig(s.Config)
	}
}
