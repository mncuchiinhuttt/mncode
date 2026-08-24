package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mncode/pkg/agent"
)

type UsageDay struct {
	Date     string `json:"date"`
	Tokens   int64  `json:"tokens"`
	Sessions int64  `json:"sessions"`
}

type UsageSummary struct {
	TotalTokens    int64 `json:"totalTokens"`
	InputTokens    int64 `json:"inputTokens"`
	OutputTokens   int64 `json:"outputTokens"`
	ThinkingTokens int64 `json:"thinkingTokens"`
	TotalSessions  int64 `json:"totalSessions"`
	RecordsCount   int64 `json:"recordsCount"`
}

type UsageStatsResponse struct {
	Success    bool         `json:"success"`
	Summary    UsageSummary `json:"summary"`
	DailyUsage []UsageDay   `json:"dailyUsage"`
}

// FetchUsageStats reads the authenticated user's daily token usage from mncode-web.
func FetchUsageStats(s *agent.Session) (*UsageStatsResponse, error) {
	key := s.Config.GetTelemetryKey()
	if key == "" {
		return nil, fmt.Errorf("no sync key configured")
	}

	request, err := http.NewRequest("GET", s.Config.GetWebBaseURL()+"/api/user/stats", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("server returned %d", response.StatusCode)
	}

	var result UsageStatsResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
