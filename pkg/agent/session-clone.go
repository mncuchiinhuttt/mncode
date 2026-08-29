package agent

import (
	"encoding/json"
	"fmt"
	"mncode/pkg/accounts"
	"mncode/pkg/config"
	"mncode/pkg/provider"
)

func cloneSessionConfig(source *config.Config) (*config.Config, error) {
	if source == nil {
		return nil, fmt.Errorf("session config is required")
	}
	data, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	var clone config.Config
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, err
	}
	return &clone, nil
}

func isolatedSubagentProvider(parent *Session, cfg *config.Config) (provider.Provider, error) {
	if cfg.APIKey == "" && parent != nil && parent.Accounts != nil {
		accountType := accountTypeForConfig(cfg.Provider)
		if accountType != "" {
			if account := parent.Accounts.GetActiveAccount(accountType); account != nil {
				cfg.APIKey = account.AccessToken
			}
		}
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("no credential available for subagent provider %s", cfg.Provider)
	}
	return provider.NewProvider(cfg)
}

func accountTypeForConfig(providerType config.ProviderType) accounts.AccountProvider {
	switch providerType {
	case config.ProviderAntigravity:
		return accounts.ProviderTypeAntigravity
	case config.ProviderOpenAI:
		return accounts.ProviderTypeOpenAI
	case config.ProviderOpenRouter:
		return accounts.ProviderTypeOpenRouter
	case config.ProviderAnthropic:
		return accounts.ProviderTypeAnthropic
	case config.ProviderGemini:
		return accounts.ProviderTypeGemini
	case config.ProviderOpenCode:
		return accounts.ProviderTypeOpenCode
	default:
		return ""
	}
}
