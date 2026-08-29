package stats

import "time"

// UsageSummary holds aggregated token metrics.
//
// TotalTokens includes input, output, and thinking tokens. ThinkingTokens is
// kept separately because providers may bill or report it independently.
type UsageSummary struct {
	InputTokens    int64 `json:"inputTokens"`
	OutputTokens   int64 `json:"outputTokens"`
	ThinkingTokens int64 `json:"thinkingTokens"`
	TotalTokens    int64 `json:"totalTokens"`
	Requests       int64 `json:"requests"`
}

// TokenRecord represents an individual usage log entry.
type TokenRecord struct {
	Timestamp      time.Time `json:"timestamp"`
	Date           string    `json:"date"`  // YYYY-MM-DD (UTC)
	Month          string    `json:"month"` // YYYY-MM (UTC)
	Model          string    `json:"model"`
	AccountID      string    `json:"accountId"`
	InputTokens    int64     `json:"inputTokens"`
	OutputTokens   int64     `json:"outputTokens"`
	ThinkingTokens int64     `json:"thinkingTokens"`
	TotalTokens    int64     `json:"totalTokens"`
}

// UsageStore represents persistent storage structure in ~/.mncode/usage.json
type UsageStore struct {
	Daily    map[string]*UsageSummary `json:"daily"`    // "2026-08-21" -> summary
	Monthly  map[string]*UsageSummary `json:"monthly"`  // "2026-08" -> summary
	ByModel  map[string]*UsageSummary `json:"byModel"`  // "gemini-2.5-pro" -> summary
	Lifetime *UsageSummary            `json:"lifetime"` // All-time total
	History  []TokenRecord            `json:"history"`  // Recent records
}
