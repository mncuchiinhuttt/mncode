package ui

import "mncode/pkg/agent"

type SettingType int

const (
	SettingTypeBool SettingType = iota
	SettingTypeChoice
	SettingTypeModel
	SettingTypeEffort
	SettingTypeWorkflow
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
		{Key: "auto_compact", Label: "Auto-compact", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "continue_at_limit", Label: "Continue automatically at usage limit", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "switch_model_flagged", Label: "Switch models when a message is flagged", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "show_tips", Label: "Show tips", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "reduce_motion", Label: "Reduce motion", Type: SettingTypeBool, DefaultVal: "false"},
		{Key: "thinking_mode", Label: "Thinking mode", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "prompt_suggestions", Label: "Prompt suggestions", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "session_recap", Label: "Session recap", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "rewind_code", Label: "Rewind code (checkpoints)", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "dynamic_workflows", Label: "Dynamic workflows", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "ultracode_trigger", Label: "Ultracode keyword trigger", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "workflow_size", Label: "Dynamic workflow size", Type: SettingTypeChoice, DefaultVal: "medium (default)", Choices: []string{"small", "medium (default)", "large", "full"}},
		{Key: "artifacts", Label: "Artifacts", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "verbose_output", Label: "Verbose output", Type: SettingTypeBool, DefaultVal: "false"},
		{Key: "terminal_progress", Label: "Terminal progress bar", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "show_turn_duration", Label: "Show turn duration", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "default_permission", Label: "Default permission mode", Type: SettingTypeChoice, DefaultVal: "Manual (Ask)", Choices: []string{"Manual (Ask)", "Auto-Approve", "Bypass Permissions"}},
		{Key: "worktree_base", Label: "Worktree base ref", Type: SettingTypeChoice, DefaultVal: "fresh", Choices: []string{"fresh", "main", "current"}},
		{Key: "auto_mode_plan", Label: "Use auto mode during plan", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "respect_gitignore", Label: "Respect .gitignore in file picker", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "skip_copy_picker", Label: "Skip the /copy picker", Type: SettingTypeBool, DefaultVal: "false"},
		{Key: "copy_on_select", Label: "Copy on select", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "auto_scroll", Label: "Auto-scroll", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "open_agents_default", Label: "Open agents view by default", Type: SettingTypeBool, DefaultVal: "false"},
		{Key: "left_opens_agents", Label: "← opens agents", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "auto_update_channel", Label: "Auto-update channel", Type: SettingTypeChoice, DefaultVal: "latest", Choices: []string{"latest", "beta", "stable"}},
		{Key: "theme", Label: "Theme", Type: SettingTypeChoice, DefaultVal: "Pastel Pink", Choices: []string{"Pastel Pink", "Dark mode", "Light mode", "Cyberpunk", "Monokai", "Tokyo Night"}},
		{Key: "diff_style", Label: "Code diff highlight style", Type: SettingTypeChoice, DefaultVal: "Full-line Background", Choices: []string{"Full-line Background", "Syntax Text Only"}},
		{Key: "local_notifications", Label: "Local notifications", Type: SettingTypeChoice, DefaultVal: "Auto", Choices: []string{"Auto", "Always", "Never"}},
		{Key: "push_actions_required", Label: "Push when actions required", Type: SettingTypeBool, DefaultVal: "false"},
		{Key: "language", Label: "Language", Type: SettingTypeChoice, DefaultVal: "Default (English)", Choices: []string{"Default (English)", "Vietnamese", "Japanese", "Chinese", "Spanish", "French", "German"}},
		{Key: "brainrot_mode", Label: "Brainrot Mode (Gen Z / Sigma / Max Rizz)", Type: SettingTypeBool, DefaultVal: "false"},
		{Key: "troll_status", Label: "Funny / Troll status messages", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "editor_mode", Label: "Editor mode", Type: SettingTypeChoice, DefaultVal: "normal", Choices: []string{"normal", "vim", "emacs"}},
		{Key: "question_timeout", Label: "Question auto-continue timeout", Type: SettingTypeChoice, DefaultVal: "never", Choices: []string{"never", "30s", "1m", "5m"}},
		{Key: "show_last_response_editor", Label: "Show last response in external editor", Type: SettingTypeBool, DefaultVal: "false"},
		{Key: "show_branch_name", Label: "Show current branch name", Type: SettingTypeBool, DefaultVal: "true"},
		{Key: "context_window", Label: "Context window size", Type: SettingTypeChoice, DefaultVal: "200K", Choices: []string{"200K", "300K", "500K", "1M"}},
		{Key: "model", Label: "Model", Type: SettingTypeModel, DefaultVal: "gemini-3.7-flash-high"},
		{Key: "auto_connect_ide", Label: "Auto-connect to IDE (external terminal)", Type: SettingTypeBool, DefaultVal: "false"},
		{Key: "chrome_default", Label: "Claude in Chrome enabled by default", Type: SettingTypeBool, DefaultVal: "false"},
		{Key: "remote_control", Label: "Enable Remote Control for all sessions", Type: SettingTypeChoice, DefaultVal: "default", Choices: []string{"default", "always", "never"}},
		{Key: "dialog_expiry", Label: "Dialog expiry", Type: SettingTypeChoice, DefaultVal: "default", Choices: []string{"default", "never", "1h", "24h"}},
		{Key: "messages_other_sessions", Label: "Messages from your other sessions", Type: SettingTypeChoice, DefaultVal: "default", Choices: []string{"default", "sync", "off"}},
	}
}

func GetSettingValue(s *agent.Session, def SettingDef) string {
	if def.Type == SettingTypeModel {
		return s.Config.Model
	}
	return s.Config.GetSetting(def.Key, def.DefaultVal)
}

func ToggleOrCycleSetting(s *agent.Session, def SettingDef) {
	if def.Type == SettingTypeBool {
		cur := s.Config.GetSetting(def.Key, def.DefaultVal)
		if cur == "true" {
			s.Config.SetSetting(def.Key, "false")
		} else {
			s.Config.SetSetting(def.Key, "true")
		}
		if def.Key == "verbose_output" {
			s.Config.Verbose = s.Config.GetSetting("verbose_output", "false") == "true"
		}
		return
	}

	if def.Type == SettingTypeChoice && len(def.Choices) > 0 {
		cur := s.Config.GetSetting(def.Key, def.DefaultVal)
		nextIdx := 0
		for i, c := range def.Choices {
			if c == cur {
				nextIdx = (i + 1) % len(def.Choices)
				break
			}
		}
		s.Config.SetSetting(def.Key, def.Choices[nextIdx])
	}
}
