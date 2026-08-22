package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"mncode/pkg/agent"
	"mncode/pkg/config"
	"mncode/pkg/stats"
)

type TelemetryPayload struct {
	ClientVersion  string         `json:"clientVersion"`
	OS             string         `json:"os"`
	Arch           string         `json:"arch"`
	SessionCount   int            `json:"sessionCount"`
	InputTokens    int            `json:"inputTokens"`
	OutputTokens   int            `json:"outputTokens"`
	TotalTokens    int            `json:"totalTokens"`
	ThinkingTokens int            `json:"thinkingTokens"`
	Models         map[string]int `json:"models"`
	Tools          map[string]int `json:"tools"`
	Date           string         `json:"date"`
}

// HandleSyncCommand processes /sync slash command
func HandleSyncCommand(parts []string, s *agent.Session) {
	if len(parts) > 1 {
		sub := strings.ToLower(parts[1])
		if sub == "key" && len(parts) > 2 {
			key := strings.TrimSpace(parts[2])
			s.Config.TelemetryKey = key
			s.Config.SetSetting("telemetry_key", key)
			_ = config.SaveConfig(s.Config)
			prefix := key
			if len(prefix) > 16 {
				prefix = prefix[:16] + "..."
			}
			fmt.Printf("\n%s Telemetry API Sync Key set: %s\n\n", BoldGreen("[Sync]"), BoldCyan(prefix))
			return
		}

		if sub == "url" && len(parts) > 2 {
			url := strings.TrimSpace(parts[2])
			s.Config.TelemetryURL = url
			s.Config.SetSetting("telemetry_url", url)
			_ = config.SaveConfig(s.Config)
			fmt.Printf("\n%s Telemetry Sync URL set to: %s\n\n", BoldGreen("[Sync]"), BoldCyan(url))
			return
		}
	}

	key := s.Config.GetTelemetryKey()
	if key == "" {
		fmt.Printf("\n%s Telemetry Sync Key not configured!\n", BoldYellow("[Sync Notice]"))
		fmt.Println("  1. Create an account on the mncode Web Dashboard.")
		fmt.Println("  2. Copy your API Key (`mnc_live_...`).")
		fmt.Printf("  3. Run: %s\n\n", BoldCyan("/sync key <your_key>"))
		return
	}

	url := s.Config.GetTelemetryURL()
	fmt.Printf("\n%s Pushing session telemetry to cloud (%s)...\n", BoldCyan("[Sync]"), GrayText(url))

	totalTokens, err := pushTelemetry(s, key, url)
	if err != nil {
		fmt.Printf("%s %v\n\n", BoldRed("[Sync Error]"), err)
		return
	}

	s.Config.SetSetting("last_telemetry_sync_date", time.Now().Format("2006-01-02"))
	_ = config.SaveConfig(s.Config)

	fmt.Printf("%s Telemetry pushed successfully! (Tokens: %s · Status: 200 OK)\n\n",
		BoldGreen("[Sync Success]"), BoldCyan(formatTokens(totalTokens)))
}

// pushTelemetry builds today's usage payload and POSTs it to the telemetry endpoint.
// Returns the total token count pushed, for display by the caller.
func pushTelemetry(s *agent.Session, key, url string) (int, error) {
	tracker := stats.NewTracker()
	today := tracker.GetToday()

	models := map[string]int{}
	if s.Config.Model != "" {
		models[s.Config.Model] = 1
	}

	toolsMap := map[string]int{}
	for _, t := range s.History {
		if t.Role == "assistant" {
			toolsMap["assistant_turns"]++
		}
	}

	payload := TelemetryPayload{
		ClientVersion:  config.CurrentVersion,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		SessionCount:   1,
		InputTokens:    int(today.InputTokens),
		OutputTokens:   int(today.OutputTokens),
		TotalTokens:    int(today.TotalTokens),
		ThinkingTokens: 0,
		Models:         models,
		Tools:          toolsMap,
		Date:           time.Now().Format("2006-01-02"),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to encode telemetry: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("connection failed: %w (make sure mncode-web is running at %s)", err, url)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return payload.TotalTokens, nil
}

// MaybeAutoSyncDaily silently pushes today's telemetry once per calendar day,
// if a sync key is configured. Meant to be called in a background goroutine
// at startup so it never delays the REPL; failures are swallowed (the user
// can still run /sync manually to see the real error).
func MaybeAutoSyncDaily(s *agent.Session) {
	key := s.Config.GetTelemetryKey()
	if key == "" {
		return
	}

	today := time.Now().Format("2006-01-02")
	if s.Config.GetSetting("last_telemetry_sync_date", "") == today {
		return
	}

	if _, err := pushTelemetry(s, key, s.Config.GetTelemetryURL()); err != nil {
		return
	}

	s.Config.SetSetting("last_telemetry_sync_date", today)
	_ = config.SaveConfig(s.Config)
}
