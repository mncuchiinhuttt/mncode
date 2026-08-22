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
)

type FeedbackPayload struct {
	Message       string `json:"message"`
	Category      string `json:"category"`
	ClientVersion string `json:"clientVersion"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
}

// HandleFeedbackCommand processes /feedback <message>
func HandleFeedbackCommand(parts []string, s *agent.Session) {
	if len(parts) < 2 {
		fmt.Printf("\n%s Usage: %s\n", BoldYellow("[Feedback]"), BoldCyan("/feedback <your message>"))
		fmt.Println("  Example: /feedback The /workflow ultra progress bar overlaps on narrow terminals")
		fmt.Println()
		return
	}

	key := s.Config.GetTelemetryKey()
	if key == "" {
		fmt.Printf("\n%s You need a Sync Key to submit feedback.\n", BoldYellow("[Feedback Notice]"))
		fmt.Println("  1. Create an account on the mncode Web Dashboard.")
		fmt.Println("  2. Copy your API Key (`mnc_live_...`).")
		fmt.Printf("  3. Run: %s\n\n", BoldCyan("/sync key <your_key>"))
		return
	}

	message := strings.TrimSpace(strings.Join(parts[1:], " "))
	url := s.Config.GetWebBaseURL() + "/api/feedback"

	fmt.Printf("\n%s Sending feedback...\n", BoldCyan("[Feedback]"))

	if err := postFeedback(url, key, message); err != nil {
		fmt.Printf("%s %v\n\n", BoldRed("[Feedback Error]"), err)
		return
	}

	fmt.Printf("%s Thanks! Your feedback has been recorded.\n\n", BoldGreen("[Feedback Sent]"))
}

func postFeedback(url, key, message string) error {
	payload := FeedbackPayload{
		Message:       message,
		Category:      "general",
		ClientVersion: config.CurrentVersion,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode feedback: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w (make sure mncode-web is running at %s)", err, url)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
