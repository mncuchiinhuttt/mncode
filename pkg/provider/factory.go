package provider

import (
	"fmt"
	"mncode/pkg/config"
	"strings"
)

// NewProvider creates a Provider based on configuration
func NewProvider(cfg *config.Config) (Provider, error) {
	if cfg.Provider == config.ProviderOpenCode {
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = "public"
		}
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://opencode.ai/zen/v1"
		}
		return NewOpenCodeProvider(apiKey, baseURL), nil
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("no API key found. Please set ANTHROPIC_API_KEY, OPENROUTER_API_KEY, GEMINI_API_KEY, or OPENAI_API_KEY in .env or environment")
	}

	if cfg.Provider == config.ProviderAntigravity || strings.HasPrefix(cfg.APIKey, "ya29.") {
		return NewAntigravityProvider(cfg.APIKey, cfg.BaseURL), nil
	}

	switch cfg.Provider {
	case config.ProviderAnthropic:
		return NewAnthropicProvider(cfg.APIKey, cfg.BaseURL), nil
	case config.ProviderOpenRouter, config.ProviderOpenAI:
		return NewOpenAIProvider(cfg.APIKey, cfg.BaseURL), nil
	case config.ProviderGemini:
		return NewGeminiProvider(cfg.APIKey, cfg.BaseURL), nil
	default:
		if strings.HasPrefix(cfg.APIKey, "sk-ant-") {
			return NewAnthropicProvider(cfg.APIKey, cfg.BaseURL), nil
		}
		return NewAntigravityProvider(cfg.APIKey, cfg.BaseURL), nil
	}
}
