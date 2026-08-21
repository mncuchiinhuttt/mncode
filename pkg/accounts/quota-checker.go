package accounts

import (
	"time"
)

type ModelQuota struct {
	ModelID             string  `json:"model_id"`
	DisplayName         string  `json:"display_name"`
	RemainingPercentage float64 `json:"remaining_percentage"`
	ResetTime           string  `json:"reset_time"`
	ResetInStr          string  `json:"reset_in_str"`
}

type AccountQuotaInfo struct {
	AccountID       string
	Provider        AccountProvider
	Status          string
	IsHealthy       bool
	LatencyMs       int64
	Tier            string
	ProjectID       string
	ModelQuotas     []ModelQuota
	AvailableModels []string
	MaxContext      int
	RPMRemaining    string
	TPMRemaining    string
	ExpiresInStr    string
	ErrorMessage    string
	LastChecked     time.Time
}

// CheckAccountQuota probes the provider API to check account status, model availability, and quota
func CheckAccountQuota(store *Store, acc *Account) *AccountQuotaInfo {
	info := &AccountQuotaInfo{
		AccountID:   acc.Email,
		Provider:    acc.Provider,
		LastChecked: time.Now(),
	}
	if info.AccountID == "" {
		info.AccountID = acc.ID
	}

	start := time.Now()

	switch acc.Provider {
	case ProviderTypeAntigravity, ProviderTypeGemini:
		checkAntigravityQuota(store, acc, info)
	case ProviderTypeCodex, ProviderTypeOpenAI:
		checkOpenAIQuota(acc, info)
	case "opencode":
		checkOpenCodeQuota(acc, info)
	}

	info.LatencyMs = time.Since(start).Milliseconds()
	return info
}
