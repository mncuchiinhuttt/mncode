package config

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
	Provider        ProviderType   `json:"provider"`
	Model           string         `json:"model"`
	APIKey          string         `json:"apiKey"`
	BaseURL         string         `json:"baseUrl"`
	ThinkingBudget  int            `json:"thinkingBudget"`
	MaxTokens       int            `json:"maxTokens"`
	Temperature     float64        `json:"temperature"`
	AutoApprove     bool           `json:"autoApprove"`
	PermissionMode  PermissionMode `json:"permissionMode"`
	Verbose         bool           `json:"verbose"`
	WorkspaceDir    string         `json:"workspaceDir"`
	ClaudeDir       string         `json:"claudeDir"`
	CodingLevel     int            `json:"codingLevel"`
	Effort          string         `json:"effort"`
	Workflow        string         `json:"workflow"`
	ThinkingLang    string            `json:"thinkingLang"`
	ResponseLang    string            `json:"responseLang"`
	Settings        map[string]string `json:"settings,omitempty"`
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
