package config

import (
	"os"
	"strings"
)

// ProviderType represents the LLM provider
type ProviderType string

const (
	ProviderAnthropic   ProviderType = "anthropic"
	ProviderOpenAI      ProviderType = "openai"
	ProviderGemini      ProviderType = "gemini"
	ProviderOpenRouter  ProviderType = "openrouter"
	ProviderOpenCode    ProviderType = "opencode"
	ProviderAntigravity ProviderType = "antigravity"
)

type PermissionMode string

const (
	PermissionModeAsk    PermissionMode = "ask"
	PermissionModeAuto   PermissionMode = "auto"
	PermissionModeBypass PermissionMode = "bypass"
	PermissionModePlan   PermissionMode = "plan"
)

// Config represents the application configuration
type Config struct {
	Provider       ProviderType      `json:"provider"`
	Model          string            `json:"model"`
	APIKey         string            `json:"apiKey"`
	BaseURL        string            `json:"baseUrl"`
	ThinkingBudget int               `json:"thinkingBudget"`
	MaxTokens      int               `json:"maxTokens"`
	Temperature    float64           `json:"temperature"`
	AutoApprove    bool              `json:"autoApprove"`
	PermissionMode PermissionMode    `json:"permissionMode"`
	Verbose        bool              `json:"verbose"`
	WorkspaceDir   string            `json:"workspaceDir"`
	ClaudeDir      string            `json:"claudeDir"`
	CodingLevel    int               `json:"codingLevel"`
	Effort         string            `json:"effort"`
	Workflow       string            `json:"workflow"`
	ContextWindow  string            `json:"contextWindow,omitempty"`
	ThinkingLang   string            `json:"thinkingLang"`
	ResponseLang   string            `json:"responseLang"`
	TelemetryKey   string            `json:"telemetryKey,omitempty"`
	TelemetryURL   string            `json:"telemetryUrl,omitempty"`
	Settings       map[string]string `json:"settings,omitempty"`
}

// GetWebBaseURL returns the origin of the mncode web app (dashboard, sync,
// feedback, login all hang off this). Resolution order: explicit
// "/config web_base_url <url>" setting, then the MNCODE_WEB_URL env var
// (so a distributed binary can be built/run pointed at production without
// every user having to configure it by hand), then localhost for local dev.
func (c *Config) GetWebBaseURL() string {
	if base := c.GetSetting("web_base_url", ""); base != "" {
		return strings.TrimRight(base, "/")
	}
	if envURL := os.Getenv("MNCODE_WEB_URL"); envURL != "" {
		return strings.TrimRight(envURL, "/")
	}
	return "http://localhost:3000"
}

func (c *Config) GetTelemetryURL() string {
	if c.TelemetryURL != "" {
		return c.TelemetryURL
	}
	if url := c.GetSetting("telemetry_url", ""); url != "" {
		return url
	}
	return c.GetWebBaseURL() + "/api/telemetry/sync"
}

func (c *Config) GetTelemetryKey() string {
	if c.TelemetryKey != "" {
		return c.TelemetryKey
	}
	return c.GetSetting("telemetry_key", "")
}

func (c *Config) GetContextWindowTokens() int {
	cw := strings.ToLower(c.ContextWindow)
	if cw == "" {
		cw = strings.ToLower(c.GetSetting("context_window", "200k"))
	}
	switch cw {
	case "300k", "300000":
		return 300000
	case "500k", "500000":
		return 500000
	case "1m", "1000k", "1000000":
		return 1000000
	default:
		return 200000
	}
}

func (c *Config) GetContextWindowLabel() string {
	tokens := c.GetContextWindowTokens()
	switch tokens {
	case 300000:
		return "300K"
	case 500000:
		return "500K"
	case 1000000:
		return "1M"
	default:
		return "200K"
	}
}

func (c *Config) GetSetting(key, def string) string {
	if c.Settings == nil {
		return def
	}
	if v, ok := c.Settings[key]; ok && v != "" {
		return v
	}
	return def
}

func (c *Config) SetSetting(key, val string) {
	if c.Settings == nil {
		c.Settings = make(map[string]string)
	}
	c.Settings[key] = val
}

// DefaultConfig returns the default configuration values
func DefaultConfig() *Config {
	return &Config{
		Provider:       ProviderAnthropic,
		Model:          "claude-3-7-sonnet-20250219",
		ThinkingBudget: 8192,
		MaxTokens:      8192,
		Temperature:    1.0,
		AutoApprove:    false,
		Verbose:        false,
		WorkspaceDir:   ".",
		ClaudeDir:      ".claude",
		CodingLevel:    -1,
		Effort:         "high",
		Workflow:       "auto",
	}
}

// CKConfig represents .claude/.ck.json config
type CKConfig struct {
	CodingLevel int `json:"codingLevel"`
	Plan        struct {
		NamingFormat string `json:"namingFormat"`
		DateFormat   string `json:"dateFormat"`
		ReportsDir   string `json:"reportsDir"`
	} `json:"plan"`
	Paths struct {
		Docs  string `json:"docs"`
		Plans string `json:"plans"`
	} `json:"paths"`
	Locale struct {
		ThinkingLanguage *string `json:"thinkingLanguage"`
		ResponseLanguage *string `json:"responseLanguage"`
	} `json:"locale"`
	Gemini struct {
		Model string `json:"model"`
	} `json:"gemini"`
}
