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
	ProviderCustom      ProviderType = "custom"
)

const (
	APIFormatAnthropic       = "anthropic-messages"
	APIFormatChatCompletions = "chat-completions"
	APIFormatResponses       = "responses"
)

type CustomModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextWindow int    `json:"contextWindow,omitempty"`
}

type CustomProvider struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	BaseURL   string        `json:"baseUrl"`
	APIFormat string        `json:"apiFormat"`
	APIKey    string        `json:"apiKey,omitempty"`
	Models    []CustomModel `json:"models,omitempty"`
}

type PermissionMode string

const (
	PermissionModeAsk    PermissionMode = "ask"
	PermissionModeAuto   PermissionMode = "auto"
	PermissionModeBypass PermissionMode = "bypass"
	PermissionModePlan   PermissionMode = "plan"
)

// Config represents the application configuration
type Config struct {
	Provider         ProviderType              `json:"provider"`
	CustomProviderID string                    `json:"customProviderId,omitempty"`
	Model            string                    `json:"model"`
	APIKey           string                    `json:"apiKey"`
	BaseURL          string                    `json:"baseUrl"`
	ThinkingBudget   int                       `json:"thinkingBudget"`
	MaxTokens        int                       `json:"maxTokens"`
	Temperature      float64                   `json:"temperature"`
	AutoApprove      bool                      `json:"autoApprove"`
	PermissionMode   PermissionMode            `json:"permissionMode"`
	Verbose          bool                      `json:"verbose"`
	WorkspaceDir     string                    `json:"workspaceDir"`
	ClaudeDir        string                    `json:"claudeDir"`
	CodingLevel      int                       `json:"codingLevel"`
	Effort           string                    `json:"effort"`
	Workflow         string                    `json:"workflow"`
	ContextWindow    string                    `json:"contextWindow,omitempty"`
	ThinkingLang     string                    `json:"thinkingLang"`
	ResponseLang     string                    `json:"responseLang"`
	TelemetryKey     string                    `json:"telemetryKey,omitempty"`
	TelemetryURL     string                    `json:"telemetryUrl,omitempty"`
	OpenCodeAPIKey   string                    `json:"opencodeApiKey,omitempty"`
	SearchEngine     string                    `json:"searchEngine,omitempty"`
	BraveAPIKey      string                    `json:"braveApiKey,omitempty"`
	TavilyAPIKey     string                    `json:"tavilyApiKey,omitempty"`
	CustomProviders  map[string]CustomProvider `json:"customProviders,omitempty"`
	Settings         map[string]string         `json:"settings,omitempty"`
}

// GetOpenCodeAPIKey returns configured OpenCode API key from config, settings, or env
func (c *Config) GetOpenCodeAPIKey() string {
	if c.OpenCodeAPIKey != "" {
		return c.OpenCodeAPIKey
	}
	if key := c.GetSetting("opencode_api_key", ""); key != "" {
		return key
	}
	if envKey := os.Getenv("OPENCODE_API_KEY"); envKey != "" {
		return envKey
	}
	if c.Provider == ProviderOpenCode && c.APIKey != "" && c.APIKey != "public" {
		return c.APIKey
	}
	return ""
}

// GetSearchEngine returns configured search engine (auto, antigravity, brave, tavily, duckduckgo).
func (c *Config) GetSearchEngine() string {
	if c == nil {
		return "auto"
	}
	engine := c.SearchEngine
	if engine == "" {
		engine = c.GetSetting("search_engine", "")
	}
	if engine == "" {
		engine = os.Getenv("SEARCH_ENGINE")
	}
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "antigravity", "google":
		return "antigravity"
	case "brave":
		return "brave"
	case "tavily":
		return "tavily"
	case "duckduckgo", "ddg":
		return "duckduckgo"
	default:
		return "auto"
	}
}

// GetBraveAPIKey returns configured Brave Search API key from config, settings, or env.
func (c *Config) GetBraveAPIKey() string {
	if c == nil {
		return strings.TrimSpace(os.Getenv("BRAVE_API_KEY"))
	}
	if key := strings.TrimSpace(c.BraveAPIKey); key != "" {
		return key
	}
	if key := strings.TrimSpace(c.GetSetting("brave_api_key", "")); key != "" {
		return key
	}
	return strings.TrimSpace(os.Getenv("BRAVE_API_KEY"))
}

// GetTavilyAPIKey returns configured Tavily Search API key from config, settings, or env.
func (c *Config) GetTavilyAPIKey() string {
	if c == nil {
		return strings.TrimSpace(os.Getenv("TAVILY_API_KEY"))
	}
	if key := strings.TrimSpace(c.TavilyAPIKey); key != "" {
		return key
	}
	if key := strings.TrimSpace(c.GetSetting("tavily_api_key", "")); key != "" {
		return key
	}
	return strings.TrimSpace(os.Getenv("TAVILY_API_KEY"))
}

func (c *Config) GetCustomProvider(id string) (CustomProvider, bool) {
	if c == nil || c.CustomProviders == nil {
		return CustomProvider{}, false
	}
	provider, ok := c.CustomProviders[id]
	return provider, ok
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
	return "https://mncode.mncuchiinhuttt.dev"
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
		Provider:        ProviderAnthropic,
		Model:           "claude-3-7-sonnet-20250219",
		ThinkingBudget:  8192,
		MaxTokens:       8192,
		Temperature:     1.0,
		AutoApprove:     false,
		Verbose:         false,
		WorkspaceDir:    ".",
		ClaudeDir:       ".claude",
		CodingLevel:     -1,
		Effort:          "high",
		Workflow:        "auto",
		SearchEngine:    "auto",
		CustomProviders: make(map[string]CustomProvider),
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
