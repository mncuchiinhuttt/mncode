package ui

import (
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"
)

type ModelChoice struct {
	ID          string
	Name        string
	Provider    config.ProviderType
	ProviderID  string
	Tag         string
	Description string
}

var curatedModels = []ModelChoice{
	// --- Antigravity Quota Models ---
	{
		ID:          "claude-sonnet-4-6",
		Name:        "Claude Sonnet 4.6 (Thinking)",
		Provider:    config.ProviderGemini,
		Tag:         "[Antigravity]",
		Description: "Claude Sonnet thinking model routed via Antigravity backend",
	},
	{
		ID:          "claude-opus-4-6-thinking",
		Name:        "Claude Opus 4.6 (Thinking)",
		Provider:    config.ProviderGemini,
		Tag:         "[Antigravity]",
		Description: "Claude Opus reasoning flagship routed via Antigravity backend",
	},
	{
		ID:          "gemini-3.7-flash-high",
		Name:        "Gemini 3.7 Flash (High)",
		Provider:    config.ProviderGemini,
		Tag:         "[Antigravity]",
		Description: "Gemini 3.7 Flash high reasoning tier on Antigravity",
	},
	{
		ID:          "gemini-3.6-flash-high",
		Name:        "Gemini 3.6 Flash (High)",
		Provider:    config.ProviderGemini,
		Tag:         "[Antigravity]",
		Description: "Gemini 3.6 Flash high thinking level on Antigravity",
	},
	{
		ID:          "gemini-pro-agent",
		Name:        "Gemini 3.1 Pro (High)",
		Provider:    config.ProviderGemini,
		Tag:         "[Antigravity]",
		Description: "Gemini Pro Agent high reasoning model on Antigravity",
	},
	{
		ID:          "gpt-oss-120b-medium",
		Name:        "GPT-OSS 120B (Medium)",
		Provider:    config.ProviderGemini,
		Tag:         "[Antigravity]",
		Description: "GPT OSS 120B medium thinking tier on Antigravity",
	},
	{
		ID:          "gemini-3.1-flash-image",
		Name:        "Gemini 3.1 Flash (Image)",
		Provider:    config.ProviderGemini,
		Tag:         "[Antigravity]",
		Description: "Multimodal image generation model on Antigravity",
	},
	{
		ID:          "gemini-2.5-pro",
		Name:        "Gemini 2.5 Pro",
		Provider:    config.ProviderGemini,
		Tag:         "[Antigravity]",
		Description: "Google flagship 1M context coding & deep reasoning model",
	},
	{
		ID:          "gemini-2.5-flash",
		Name:        "Gemini 2.5 Flash",
		Provider:    config.ProviderGemini,
		Tag:         "[Antigravity]",
		Description: "1M context high-speed model with minimal latency for rapid edits",
	},

	// --- OpenCode Zen Free Models & Stealth Reasoning ---
	{
		// OpenCode Zen's real slug for "Ox Alpha Free" is x-preview-f-free —
		// "ox-alpha" 401s ("Model ox-alpha is not supported"), verified live.
		ID:          "x-preview-f-free",
		Name:        "Ox Alpha (1M Free Reasoning)",
		Provider:    config.ProviderOpenCode,
		Tag:         "[OpenCode Free]",
		Description: "Stealth 1M context unlimited reasoning model for coding & agentic workflows (100% Free)",
	},
	{
		// Slug is correct (confirmed against OpenCode Zen docs) and returns a
		// distinct "model is unavailable" server_error via the anonymous public
		// key — not an auth/not-found error — so this looks like a temporarily
		// paused free-tier model upstream, not a wrong ID. Re-verify occasionally.
		ID:          "deepseek-v4-flash-free",
		Name:        "DeepSeek V4 Flash (Free, currently unavailable)",
		Provider:    config.ProviderOpenCode,
		Tag:         "[OpenCode Free]",
		Description: "OpenCode Zen Free tier DeepSeek V4 Flash model. Currently returning \"model unavailable\" upstream — try mimo-v2.5-free or x-preview-f-free instead for now.",
	},
	{
		ID:          "mimo-v2.5-free",
		Name:        "MiMo V2.5 (Free)",
		Provider:    config.ProviderOpenCode,
		Tag:         "[OpenCode Free]",
		Description: "Xiaomi MiMo V2.5 coding specialist free model via OpenCode Zen",
	},
	{
		ID:          "nemotron-3-ultra-free",
		Name:        "Nemotron 3 Ultra (Free)",
		Provider:    config.ProviderOpenCode,
		Tag:         "[OpenCode Free]",
		Description: "NVIDIA Nemotron 3 Ultra free tier model via OpenCode Zen",
	},
	{
		ID:          "nemotron-3.5-lightning-free",
		Name:        "Nemotron 3.5 Lightning (Free)",
		Provider:    config.ProviderOpenCode,
		Tag:         "[OpenCode Free]",
		Description: "NVIDIA Nemotron 3.5 Lightning fast coding model via OpenCode Zen",
	},
	{
		ID:          "hy3-free",
		Name:        "Hunyuan 3 (Free)",
		Provider:    config.ProviderOpenCode,
		Tag:         "[OpenCode Free]",
		Description: "Tencent Hunyuan 3 free tier model via OpenCode Zen",
	},

	// --- Direct Provider & API Models ---
	{
		ID:          "claude-3-7-sonnet",
		Name:        "Claude 3.7 Sonnet",
		Provider:    config.ProviderAnthropic,
		Tag:         "[Anthropic API]",
		Description: "Anthropic hybrid reasoning model for complex architecture",
	},
	{
		ID:          "claude-3-5-sonnet",
		Name:        "Claude 3.5 Sonnet",
		Provider:    config.ProviderAnthropic,
		Tag:         "[Anthropic API]",
		Description: "Standard coding specialist model via Anthropic API key",
	},
	{
		ID:          "gpt-4o",
		Name:        "GPT-4o",
		Provider:    config.ProviderOpenAI,
		Tag:         "[OpenAI API]",
		Description: "OpenAI versatile flagship model via OpenAI API key",
	},
	{
		ID:          "o3-mini",
		Name:        "o3-mini",
		Provider:    config.ProviderOpenAI,
		Tag:         "[OpenAI API]",
		Description: "OpenAI high-speed reasoning model with adjustable thinking effort",
	},
	{
		ID:          "deepseek-chat",
		Name:        "DeepSeek V3",
		Provider:    config.ProviderOpenRouter,
		Tag:         "[OpenRouter]",
		Description: "High capability open MoE model via OpenRouter",
	},
	{
		ID:          "deepseek-reasoner",
		Name:        "DeepSeek R1",
		Provider:    config.ProviderOpenRouter,
		Tag:         "[OpenRouter]",
		Description: "Deep reasoning model with transparent chain-of-thought",
	},
	{
		ID:          "custom",
		Name:        "Custom Model...",
		Provider:    "",
		Tag:         "[Manual Entry]",
		Description: "Type any custom model ID manually (e.g. gemini-3.7-flash-high, mistral-large)",
	},
}

// openAIModelCatalog mirrors the text/coding models in the official OpenAI
// catalog. Specialized audio, image, embedding, and moderation-only models
// are intentionally excluded because the coding agent uses text generation.
var openAIModelCatalog = []ModelChoice{
	{ID: "gpt-5.6", Name: "GPT-5.6", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI latest flagship alias for complex reasoning and coding"},
	{ID: "gpt-5.6-sol", Name: "GPT-5.6 Sol", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI frontier model for complex professional work"},
	{ID: "gpt-5.6-terra", Name: "GPT-5.6 Terra", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI model balancing intelligence and cost"},
	{ID: "gpt-5.6-luna", Name: "GPT-5.6 Luna", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI efficient model for high-volume workloads"},
	{ID: "gpt-5.5", Name: "GPT-5.5", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI model for coding and professional work"},
	{ID: "gpt-5.5-pro", Name: "GPT-5.5 Pro", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI higher-compute GPT-5.5 variant"},
	{ID: "gpt-5.4", Name: "GPT-5.4", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI model for coding and professional work"},
	{ID: "gpt-5.4-pro", Name: "GPT-5.4 Pro", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI higher-compute GPT-5.4 variant"},
	{ID: "gpt-5.4-mini", Name: "GPT-5.4 mini", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI efficient model for coding, computer use, and subagents"},
	{ID: "gpt-5.4-nano", Name: "GPT-5.4 nano", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI low-cost model for simple workloads"},
	{ID: "gpt-5.3-codex", Name: "GPT-5.3 Codex", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI agentic coding model"},
	{ID: "gpt-5.2", Name: "GPT-5.2", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI frontier model for professional work"},
	{ID: "gpt-5.2-pro", Name: "GPT-5.2 Pro", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI higher-compute GPT-5.2 variant"},
	{ID: "gpt-5.1", Name: "GPT-5.1", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI model for coding and agentic tasks"},
	{ID: "gpt-5", Name: "GPT-5", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI reasoning model for coding and agentic tasks"},
	{ID: "gpt-5-mini", Name: "GPT-5 mini", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI efficient reasoning model"},
	{ID: "gpt-5-nano", Name: "GPT-5 nano", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI fastest low-cost reasoning model"},
	{ID: "gpt-5-pro", Name: "GPT-5 Pro", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI higher-compute GPT-5 variant"},
	{ID: "o3-pro", Name: "o3-pro", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI higher-compute reasoning model"},
	{ID: "o3", Name: "o3", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI reasoning model for complex tasks"},
	{ID: "gpt-4.1", Name: "GPT-4.1", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI non-reasoning model for coding"},
	{ID: "gpt-4.1-mini", Name: "GPT-4.1 mini", Provider: config.ProviderOpenAI, Tag: "[OpenAI API]", Description: "OpenAI smaller, faster GPT-4.1 variant"},
	{ID: "gpt-4.1-nano", Name: "GPT-4.1 nano", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated OpenAI low-cost GPT-4.1 variant"},
	{ID: "gpt-5.2-codex", Name: "GPT-5.2 Codex", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated OpenAI long-horizon coding model"},
	{ID: "gpt-5.1-codex", Name: "GPT-5.1 Codex", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated OpenAI coding model"},
	{ID: "gpt-5.1-codex-max", Name: "GPT-5.1 Codex Max", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated OpenAI long-running Codex model"},
	{ID: "gpt-5.1-codex-mini", Name: "GPT-5.1 Codex mini", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated efficient Codex model"},
	{ID: "gpt-5-codex", Name: "GPT-5 Codex", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated OpenAI agentic coding model"},
	{ID: "codex-mini-latest", Name: "Codex mini latest", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated fast reasoning model optimized for Codex CLI"},
	{ID: "o4-mini", Name: "o4-mini", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated fast, cost-efficient reasoning model"},
	{ID: "o1-pro", Name: "o1-pro", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated higher-compute reasoning model"},
	{ID: "o1", Name: "o1", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated OpenAI reasoning model"},
	{ID: "o1-mini", Name: "o1-mini", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated smaller reasoning model"},
	{ID: "o1-preview", Name: "o1 Preview", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated preview reasoning model"},
	{ID: "gpt-4.5-preview", Name: "GPT-4.5 Preview", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated GPT-4.5 preview model"},
	{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated high-intelligence GPT-4 model"},
	{ID: "gpt-4", Name: "GPT-4", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated high-intelligence GPT model"},
	{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: config.ProviderOpenAI, Tag: "[OpenAI API · Deprecated]", Description: "Deprecated legacy GPT model"},
}

// GetAvailableModels filters models to only show authenticated providers, free tiers, and current model
func GetAvailableModels(s *agent.Session) []ModelChoice {
	hasAntigravity := false
	hasCodex := false

	if s != nil && s.Accounts != nil {
		for _, acc := range s.Accounts.Accounts {
			if acc.Provider == "antigravity" || acc.Provider == "gemini" {
				hasAntigravity = true
			}
			if acc.Provider == "codex" || acc.Provider == "openai" {
				hasCodex = true
			}
		}
	}

	hasAnthropic := os.Getenv("ANTHROPIC_API_KEY") != "" || (s != nil && s.Config.Provider == config.ProviderAnthropic && s.Config.APIKey != "")
	hasOpenAI := hasCodex || os.Getenv("OPENAI_API_KEY") != "" || (s != nil && s.Config.Provider == config.ProviderOpenAI && s.Config.APIKey != "")
	hasOpenRouter := os.Getenv("OPENROUTER_API_KEY") != "" || (s != nil && s.Config.Provider == config.ProviderOpenRouter && s.Config.APIKey != "")
	hasGemini := hasAntigravity || os.Getenv("GEMINI_API_KEY") != "" || (s != nil && (s.Config.Provider == config.ProviderAntigravity || s.Config.Provider == config.ProviderGemini) && s.Config.APIKey != "")

	var result []ModelChoice

	allModels := make([]ModelChoice, 0, len(curatedModels)+len(openAIModelCatalog))
	allModels = append(allModels, curatedModels...)
	allModels = append(allModels, openAIModelCatalog...)
	for _, m := range allModels {
		switch m.Tag {
		case "[Antigravity]":
			if hasGemini {
				result = append(result, m)
			}
		case "[OpenCode Free]", "[OpenRouter Free]":
			result = append(result, m)
		case "[Anthropic API]":
			if hasAnthropic {
				result = append(result, m)
			}
		case "[OpenAI API]":
			if hasOpenAI {
				result = append(result, m)
			}
		case "[OpenRouter]":
			if hasOpenRouter {
				result = append(result, m)
			}
		case "[Manual Entry]":
			result = append(result, m)
		default:
			if strings.HasPrefix(m.Tag, "[OpenAI API") && hasOpenAI {
				result = append(result, m)
			}
		}
	}

	if s != nil && s.Config.Model != "" {
		found := false
		for _, m := range result {
			if strings.EqualFold(m.ID, s.Config.Model) {
				found = true
				break
			}
		}
		if !found {
			customItem := ModelChoice{
				ID:          s.Config.Model,
				Name:        s.Config.Model,
				Provider:    s.Config.Provider,
				Tag:         "[Current]",
				Description: "Currently active configured model",
			}
			result = append([]ModelChoice{customItem}, result...)
		}
	}

	if s != nil && s.Config != nil {
		for _, custom := range s.Config.CustomProviders {
			for _, model := range custom.Models {
				if strings.TrimSpace(model.ID) == "" {
					continue
				}
				name := model.Name
				if name == "" {
					name = model.ID
				}
				result = append(result, ModelChoice{
					ID: model.ID, Name: name, Provider: config.ProviderCustom,
					ProviderID: custom.ID, Tag: "[" + custom.Name + "]",
					Description: "Custom model from " + custom.Name,
				})
			}
		}
	}

	return result
}
