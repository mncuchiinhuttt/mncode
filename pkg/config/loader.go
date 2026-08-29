package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LoadConfig loads the full configuration from environment, .env, and .claude directory
func LoadConfig(workspaceDir string) (*Config, error) {
	cfg := DefaultConfig()

	if workspaceDir != "" {
		absPath, err := filepath.Abs(workspaceDir)
		if err == nil {
			cfg.WorkspaceDir = absPath
		}
	} else {
		cwd, err := os.Getwd()
		if err == nil {
			cfg.WorkspaceDir = cwd
		}
	}

	// 1. Load .env from workspace or current dir
	_ = LoadDotEnv(filepath.Join(cfg.WorkspaceDir, ".env"))
	_ = LoadDotEnv(filepath.Join(cfg.WorkspaceDir, ".claude", ".env"))

	// 2. Locate .claude directory
	claudeDir := filepath.Join(cfg.WorkspaceDir, ".claude")
	if info, err := os.Stat(claudeDir); err == nil && info.IsDir() {
		cfg.ClaudeDir = claudeDir
	}

	// 3. Load .ck.json if present
	ckPath := filepath.Join(cfg.ClaudeDir, ".ck.json")
	if data, err := os.ReadFile(ckPath); err == nil {
		var ck CKConfig
		if err := json.Unmarshal(data, &ck); err == nil {
			cfg.CodingLevel = ck.CodingLevel
			if ck.Locale.ThinkingLanguage != nil {
				cfg.ThinkingLang = *ck.Locale.ThinkingLanguage
			}
			if ck.Locale.ResponseLanguage != nil {
				cfg.ResponseLang = *ck.Locale.ResponseLanguage
			}
		}
	}

	// 4. Resolve API keys & Provider preferences
	resolveProviderAndKeys(cfg)

	// 5. Load user saved config from ~/.mncode/config.json
	_ = LoadUserConfig(cfg)

	return cfg, nil
}

func resolveProviderAndKeys(cfg *Config) {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg.Provider = ProviderAnthropic
		cfg.APIKey = key
		if model := os.Getenv("ANTHROPIC_MODEL"); model != "" {
			cfg.Model = model
		}
	} else if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		cfg.Provider = ProviderOpenRouter
		cfg.APIKey = key
		cfg.BaseURL = "https://openrouter.ai/api/v1"
		cfg.Model = "anthropic/claude-3.7-sonnet"
		if model := os.Getenv("OPENROUTER_MODEL"); model != "" {
			cfg.Model = model
		}
	} else if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		cfg.Provider = ProviderGemini
		cfg.APIKey = key
		cfg.Model = "gemini-2.5-pro"
		if model := os.Getenv("GEMINI_MODEL"); model != "" {
			cfg.Model = model
		}
	} else if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cfg.Provider = ProviderOpenAI
		cfg.APIKey = key
		cfg.Model = "gpt-4o"
		if model := os.Getenv("OPENAI_MODEL"); model != "" {
			cfg.Model = model
		}
	}

	// Allow custom base URL override
	if base := os.Getenv("LLM_BASE_URL"); base != "" {
		cfg.BaseURL = base
	}
	if model := os.Getenv("LLM_MODEL"); model != "" {
		cfg.Model = model
	}

	if braveKey := os.Getenv("BRAVE_API_KEY"); braveKey != "" {
		cfg.BraveAPIKey = braveKey
	}
	if tavilyKey := os.Getenv("TAVILY_API_KEY"); tavilyKey != "" {
		cfg.TavilyAPIKey = tavilyKey
	}
	if engine := os.Getenv("SEARCH_ENGINE"); engine != "" {
		cfg.SearchEngine = engine
	}
}
