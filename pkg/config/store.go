package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// GetConfigFilePath returns the path to ~/.mncode/config.json
func GetConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".mncode")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// SaveConfig persists the current configuration to ~/.mncode/config.json
func SaveConfig(cfg *Config) error {
	filePath, err := GetConfigFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// LoadUserConfig loads saved settings from ~/.mncode/config.json on top of existing config
func LoadUserConfig(cfg *Config) error {
	filePath, err := GetConfigFilePath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil // file doesn't exist yet, silently ignore
	}

	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		return err
	}

	if saved.Model != "" {
		cfg.Model = saved.Model
	}
	if saved.Provider != "" {
		cfg.Provider = saved.Provider
	}
	if saved.APIKey != "" {
		cfg.APIKey = saved.APIKey
	}
	if saved.BaseURL != "" {
		if (saved.Provider == ProviderAntigravity || saved.Provider == ProviderGemini) && !strings.Contains(saved.BaseURL, "googleapis.com") {
			cfg.BaseURL = ""
		} else if saved.Provider == ProviderAnthropic && !strings.Contains(saved.BaseURL, "anthropic.com") {
			cfg.BaseURL = ""
		} else {
			cfg.BaseURL = saved.BaseURL
		}
	}
	if saved.ThinkingBudget > 0 {
		cfg.ThinkingBudget = saved.ThinkingBudget
	}
	if saved.MaxTokens > 0 {
		cfg.MaxTokens = saved.MaxTokens
	}
	if saved.Temperature > 0 {
		cfg.Temperature = saved.Temperature
	}
	if saved.Effort != "" {
		cfg.Effort = saved.Effort
	}
	if saved.Workflow != "" {
		cfg.Workflow = saved.Workflow
	}
	if saved.ThinkingLang != "" {
		cfg.ThinkingLang = saved.ThinkingLang
	}
	if saved.ResponseLang != "" {
		cfg.ResponseLang = saved.ResponseLang
	}
	if saved.PermissionMode != "" {
		cfg.PermissionMode = saved.PermissionMode
	} else if saved.AutoApprove {
		cfg.PermissionMode = PermissionModeAuto
	} else {
		cfg.PermissionMode = PermissionModeAsk
	}
	if cfg.PermissionMode == PermissionModeBypass || cfg.PermissionMode == PermissionModeAuto {
		cfg.AutoApprove = true
	} else {
		cfg.AutoApprove = false
	}
	cfg.Verbose = saved.Verbose
	if saved.OpenCodeAPIKey != "" {
		cfg.OpenCodeAPIKey = saved.OpenCodeAPIKey
	}
	if saved.CustomProviderID != "" {
		cfg.CustomProviderID = saved.CustomProviderID
	}
	if saved.CustomProviders != nil {
		cfg.CustomProviders = saved.CustomProviders
	}
	if saved.TelemetryKey != "" {
		cfg.TelemetryKey = saved.TelemetryKey
	}
	if saved.TelemetryURL != "" {
		cfg.TelemetryURL = saved.TelemetryURL
	}
	if saved.Settings != nil {
		cfg.Settings = saved.Settings
	}

	// Token saver: route through a local headroom proxy when enabled, the CLI
	// is installed, and the user has no custom base URL. Spawns the proxy on
	// demand if it is not running yet.
	if cfg.BaseURL == "" &&
		cfg.GetSetting("token_saver_headroom", "false") == "true" &&
		HeadroomInstalled() {
		cfg.BaseURL = EnsureHeadroomProxy()
	}

	return nil
}
