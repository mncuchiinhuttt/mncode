package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strconv"
	"strings"
)

// HandleConfigCommand manages get/set/list/reset for application configuration
func HandleConfigCommand(parts []string, s *agent.Session) {
	if len(parts) == 1 {
		OpenInteractiveConfigMenu(s)
		return
	}
	if len(parts) == 2 && parts[1] == "list" {
		showConfigList(s)
		return
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "get":
		if len(parts) < 3 {
			fmt.Println("Usage: /config get <key>")
			return
		}
		getConfigValue(parts[2], s)

	case "set":
		if len(parts) < 4 {
			fmt.Println("Usage: /config set <key> <value>")
			fmt.Println("Examples:")
			fmt.Println("  /config set model gemini-2.5-pro")
			fmt.Println("  /config set provider gemini")
			fmt.Println("  /config set effort max")
			fmt.Println("  /config set workflow ultra-workflow")
			fmt.Println("  /config set temperature 0.7")
			fmt.Println("  /config set max_tokens 16384")
			return
		}
		setConfigValue(parts[2], strings.Join(parts[3:], " "), s)

	case "reset":
		configPath, _ := config.GetConfigFilePath()
		_ = os.Remove(configPath)
		def := config.DefaultConfig()
		s.Config = def
		fmt.Printf("%s Configuration reset to defaults.\n", BoldGreen("[Success]"))

	default:
		// Check if user did `/config <key> <value>` shorthand
		if len(parts) == 3 {
			setConfigValue(parts[1], parts[2], s)
		} else {
			fmt.Printf("Unknown subcommand '%s'. Use /config list, /config get <key>, /config set <key> <val>, /config reset\n", sub)
		}
	}
}

func showConfigList(s *agent.Session) {
	cfg := s.Config
	configPath, _ := config.GetConfigFilePath()

	fmt.Println()
	fmt.Println(BoldCyan("MNCODE CONFIGURATION:"))
	fmt.Println()
	fmt.Printf("  %-18s %-32s\n", Bold("Setting"), Bold("Current Value"))
	fmt.Println("  -------------------------------------------------------------")
	fmt.Printf("  %-18s %s\n", "model", BoldGreen(cfg.Model))
	fmt.Printf("  %-18s %s\n", "provider", BoldCyan(string(cfg.Provider)))
	fmt.Printf("  %-18s %s\n", "effort", BoldYellow(strings.ToUpper(cfg.Effort)))
	fmt.Printf("  %-18s %s\n", "workflow", BoldCyan(strings.ToUpper(cfg.Workflow)))
	fmt.Printf("  %-18s %d tokens\n", "thinking_budget", cfg.ThinkingBudget)
	fmt.Printf("  %-18s %d tokens\n", "max_tokens", cfg.MaxTokens)
	fmt.Printf("  %-18s %.2f\n", "temperature", cfg.Temperature)
	fmt.Printf("  %-18s %v\n", "auto_approve", cfg.AutoApprove)
	fmt.Printf("  %-18s %s\n", "workspace", cfg.WorkspaceDir)
	if cfg.BaseURL != "" {
		fmt.Printf("  %-18s %s\n", "base_url", cfg.BaseURL)
	}
	if cfg.CodingLevel >= 0 {
		fmt.Printf("  %-18s %d\n", "coding_level", cfg.CodingLevel)
	}
	if cfg.ThinkingLang != "" {
		fmt.Printf("  %-18s %s\n", "thinking_lang", cfg.ThinkingLang)
	}
	if cfg.ResponseLang != "" {
		fmt.Printf("  %-18s %s\n", "response_lang", cfg.ResponseLang)
	}
	fmt.Println("  -------------------------------------------------------------")
	fmt.Printf("  Config File: %s\n", GrayText(configPath))
	fmt.Println()
	fmt.Println(GrayText("  Use '/config set <key> <value>' to modify and persist any setting."))
	fmt.Println()
}

func getConfigValue(key string, s *agent.Session) {
	cfg := s.Config
	switch strings.ToLower(key) {
	case "model":
		fmt.Printf("model = %s\n", cfg.Model)
	case "provider":
		fmt.Printf("provider = %s\n", cfg.Provider)
	case "effort":
		fmt.Printf("effort = %s\n", cfg.Effort)
	case "workflow":
		fmt.Printf("workflow = %s\n", cfg.Workflow)
	case "thinking_budget", "budget":
		fmt.Printf("thinking_budget = %d\n", cfg.ThinkingBudget)
	case "max_tokens":
		fmt.Printf("max_tokens = %d\n", cfg.MaxTokens)
	case "temperature", "temp":
		fmt.Printf("temperature = %.2f\n", cfg.Temperature)
	case "auto_approve":
		fmt.Printf("auto_approve = %v\n", cfg.AutoApprove)
	case "base_url":
		fmt.Printf("base_url = %s\n", cfg.BaseURL)
	default:
		fmt.Printf("Unknown config key '%s'.\n", key)
	}
}

func setConfigValue(key, value string, s *agent.Session) {
	cfg := s.Config
	k := strings.ToLower(key)

	switch k {
	case "model":
		cfg.Model = value
	case "provider":
		cfg.Provider = config.ProviderType(strings.ToLower(value))
		_ = s.EnsureProvider()
	case "effort":
		HandleEffortCommand([]string{"/effort", value}, s)
		return
	case "workflow":
		HandleWorkflowCommand([]string{"/workflow", value}, s)
		return
	case "thinking_budget", "budget":
		if v, err := strconv.Atoi(value); err == nil && v >= 0 {
			cfg.ThinkingBudget = v
		} else {
			fmt.Println("Invalid thinking_budget integer value.")
			return
		}
	case "max_tokens":
		if v, err := strconv.Atoi(value); err == nil && v > 0 {
			cfg.MaxTokens = v
		} else {
			fmt.Println("Invalid max_tokens integer value.")
			return
		}
	case "temperature", "temp":
		if v, err := strconv.ParseFloat(value, 64); err == nil && v >= 0.0 && v <= 2.0 {
			cfg.Temperature = v
		} else {
			fmt.Println("Invalid temperature float value (0.0 to 2.0).")
			return
		}
	case "auto_approve":
		cfg.AutoApprove = (strings.ToLower(value) == "true" || value == "1" || value == "yes")
	case "opencode_api_key", "opencode_key":
		cfg.OpenCodeAPIKey = value
		cfg.SetSetting("opencode_api_key", value)
		if cfg.Provider == config.ProviderOpenCode {
			cfg.APIKey = value
			_ = s.EnsureProvider()
		}
	case "api_key", "key":
		cfg.APIKey = value
		if cfg.Provider == config.ProviderOpenCode {
			cfg.OpenCodeAPIKey = value
			cfg.SetSetting("opencode_api_key", value)
		}
		_ = s.EnsureProvider()
	case "base_url":
		cfg.BaseURL = value
		_ = s.EnsureProvider()
	default:
		fmt.Printf("Unknown config key '%s'. Available: model, provider, effort, workflow, thinking_budget, max_tokens, temperature, auto_approve, base_url, opencode_api_key, api_key\n", key)
		return
	}

	if err := config.SaveConfig(cfg); err != nil {
		fmt.Printf("%s Failed saving to config.json: %v\n", BoldRed("[Error]"), err)
	} else {
		fmt.Printf("%s Updated %s = %s and saved to ~/.mncode/config.json\n", BoldGreen("[Success]"), Bold(key), BoldCyan(value))
	}
}
