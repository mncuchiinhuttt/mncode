package accounts

import (
	"time"
)

// AccountProvider represents the service type
type AccountProvider string

const (
	ProviderTypeAntigravity AccountProvider = "antigravity"
	ProviderTypeCodex       AccountProvider = "codex"
	ProviderTypeAnthropic   AccountProvider = "anthropic"
	ProviderTypeGemini      AccountProvider = "gemini"
	ProviderTypeOpenAI      AccountProvider = "openai"
	ProviderTypeOpenCode    AccountProvider = "opencode"
)

// Account represents a single logged-in account/credential
type Account struct {
	ID           string          `json:"id"`
	Email        string          `json:"email"`
	Provider     AccountProvider `json:"provider"`
	AccessToken  string          `json:"accessToken"`
	RefreshToken string          `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time       `json:"expiresAt"`
	IsActive     bool            `json:"isActive"`
	CooldownUntil time.Time      `json:"cooldownUntil,omitempty"`
	UsageCount   int64           `json:"usageCount"`
	LastError    string          `json:"lastError,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
}

// IsAvailable checks if the account is active, not expired, and not in cooldown
func (a *Account) IsAvailable() bool {
	if !a.IsActive {
		return false
	}
	if time.Now().Before(a.CooldownUntil) {
		return false
	}
	if !a.ExpiresAt.IsZero() && time.Now().After(a.ExpiresAt) && a.RefreshToken == "" {
		return false
	}
	return true
}

// MarkCooldown sets a temporary cooldown period when rate limited (e.g. 429)
func (a *Account) MarkCooldown(duration time.Duration, reason string) {
	a.CooldownUntil = time.Now().Add(duration)
	a.LastError = reason
}
