package agent

import (
	"bufio"
	"fmt"
	"mncode/pkg/accounts"
	"mncode/pkg/config"
	"mncode/pkg/provider"
	"os"
	"strings"
	"time"
)

// EnsureProvider verifies or dynamically initializes an LLM provider before processing prompts
func (s *Session) EnsureProvider() error {
	if s.Config.Provider == config.ProviderOpenCode {
		key := s.Config.GetOpenCodeAPIKey()
		if key == "" && s.Accounts != nil {
			for _, acc := range s.Accounts.Accounts {
				if acc.Provider == accounts.ProviderTypeOpenCode && acc.IsActive && acc.AccessToken != "" {
					key = acc.AccessToken
					break
				}
			}
		}
		if key == "" && s.Config.APIKey != "" && !strings.HasPrefix(s.Config.APIKey, "ya29.") && !strings.HasPrefix(s.Config.APIKey, "sk-ant-") {
			key = s.Config.APIKey
		}
		if key != "" {
			s.Config.APIKey = key
		}
		s.Provider = provider.NewOpenCodeProvider(key, s.Config.BaseURL)
		return nil
	}

	if s.Provider != nil {
		return nil
	}

	// 1. Try resolving from Accounts pool
	if s.Router != nil {
		if acc, err := s.Router.GetNextAccount(accounts.ProviderTypeAntigravity); err == nil && acc != nil {
			s.Config.Provider = config.ProviderAntigravity
			s.Config.APIKey = acc.AccessToken
			if s.Config.Model == "" {
				s.Config.Model = "gemini-3.7-flash-high"
			}
			ap := provider.NewAntigravityProvider(acc.AccessToken, s.Config.BaseURL)
			ap.RefreshToken = acc.RefreshToken
			targetID := acc.ID
			ap.OnTokenRefreshed = func(newTok string) {
				s.Config.APIKey = newTok
				if s.Accounts != nil {
					for _, a := range s.Accounts.Accounts {
						if a.ID == targetID {
							a.AccessToken = newTok
						}
					}
					_ = s.Accounts.Save()
				}
			}
			s.Provider = ap
			return nil
		}

		if acc, err := s.Router.GetNextAccount(accounts.ProviderTypeCodex); err == nil && acc != nil {
			s.Config.Provider = config.ProviderOpenAI
			s.Config.APIKey = acc.AccessToken
			if s.Config.Model == "" {
				s.Config.Model = "gpt-4o"
			}
			s.Provider = provider.NewOpenAIProvider(acc.AccessToken, s.Config.BaseURL)
			return nil
		}
	}

	// 2. Prompt user interactively to setup on the fly
	fmt.Println("\n" + "\033[1;33m[Notice] Chua co API Key hoac tai khoan nao duoc cau hinh.\033[0m")
	fmt.Println("Ban co the dan truc tiep API key (Anthropic/OpenAI/Gemini/OpenRouter) vao day:")
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	key, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("không thể đọc API key: %w", err)
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("vui lòng nhập API Key hoặc gõ /login antigravity để tiếp tục")
	}

	// Auto-detect provider from key prefix
	if strings.HasPrefix(key, "sk-ant-") {
		s.Config.Provider = config.ProviderAnthropic
		s.Config.APIKey = key
	} else if strings.HasPrefix(key, "AIza") {
		s.Config.Provider = config.ProviderGemini
		s.Config.APIKey = key
		s.Config.Model = "gemini-2.5-pro"
	} else if strings.HasPrefix(key, "sk-or-") {
		s.Config.Provider = config.ProviderOpenRouter
		s.Config.APIKey = key
		s.Config.BaseURL = "https://openrouter.ai/api/v1"
	} else if strings.HasPrefix(key, "ya29.") {
		s.Config.Provider = config.ProviderAntigravity
		s.Config.APIKey = key
		s.Config.Model = "gemini-3.7-flash-high"
	} else {
		s.Config.Provider = config.ProviderOpenAI
		s.Config.APIKey = key
	}

	// Save to accounts store
	if s.Accounts != nil {
		_ = s.Accounts.AddOrUpdate(&accounts.Account{
			ID:          fmt.Sprintf("key-%d", time.Now().Unix()),
			Email:       "manual-key",
			Provider:    accounts.AccountProvider(s.Config.Provider),
			AccessToken: key,
			IsActive:    true,
			CreatedAt:   time.Now(),
		})
	}

	prov, err := provider.NewProvider(s.Config)
	if err != nil {
		return err
	}

	s.Provider = prov
	fmt.Println("\033[1;32m[OK] Da thiet lap thanh cong Provider:\033[0m", s.Config.Provider)
	return nil
}
