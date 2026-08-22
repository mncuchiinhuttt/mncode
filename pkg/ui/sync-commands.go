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

	inputTokens := int(today.InputTokens)
	outputTokens := int(today.OutputTokens)
	totalTokens := int(today.TotalTokens)

	payload := TelemetryPayload{
		ClientVersion:  "0.1.0-beta",
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		SessionCount:   1,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
		TotalTokens:    totalTokens,
		ThinkingTokens: 0,
		Models:         models,
		Tools:          toolsMap,
		Date:           time.Now().Format("2006-01-02"),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("%s %v\n\n", BoldRed("[Sync Error] Failed to encode telemetry:"), err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("%s %v\n\n", BoldRed("[Sync Error] Failed to create request:"), err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s %v\n  (Make sure mncode-web is running at %s)\n\n",
			BoldRed("[Sync Error] Connection failed:"), err, url)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("%s Telemetry pushed successfully! (Tokens: %s · Status: 200 OK)\n\n",
			BoldGreen("[Sync Success]"), BoldCyan(formatTokens(payload.TotalTokens)))
	} else {
		fmt.Printf("%s Server returned %d: %s\n\n",
			BoldRed("[Sync Failed]"), resp.StatusCode, string(body))
	}
}
