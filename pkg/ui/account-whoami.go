package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mncode/pkg/agent"
)

type WhoAmIResponse struct {
	Success bool `json:"success"`
	User    struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
	IsAdmin bool `json:"isAdmin"`
}

// FetchWhoAmI resolves the web account linked to the configured sync key.
// It is shared by the CLI and desktop clients so both surfaces use the same
// mnc_live API-key identity contract.
func FetchWhoAmI(s *agent.Session) (*WhoAmIResponse, error) {
	key := s.Config.GetTelemetryKey()
	if key == "" {
		return nil, fmt.Errorf("no sync key configured")
	}

	url := s.Config.GetWebBaseURL() + "/api/keys/whoami"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var result WhoAmIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
