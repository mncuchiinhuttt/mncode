package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
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

// HandleSyncCommand processes /sync slash command.
func HandleSyncCommand(parts []string, s *agent.Session) {
	if s == nil || s.Config == nil {
		fmt.Printf("\n%s Session configuration unavailable.\n\n", BoldRed("[Sync Error]"))
		return
	}

	if len(parts) > 1 {
		sub := strings.ToLower(strings.TrimSpace(parts[1]))
		if sub == "consent" {
			granted := true
			if len(parts) > 2 {
				switch strings.ToLower(strings.TrimSpace(parts[2])) {
				case "no", "false", "0", "revoke":
					granted = false
				case "yes", "true", "1":
					granted = true
				default:
					fmt.Printf("\n%s Use %s or %s.\n\n", BoldYellow("[Sync Notice]"),
						BoldCyan("/sync consent yes"), BoldCyan("/sync consent no"))
					return
				}
			}
			s.Config.SetSetting("telemetry_sync_consent", fmt.Sprintf("%t", granted))
			if err := config.SaveConfig(s.Config); err != nil {
				fmt.Printf("\n%s Could not save sync consent: %v\n\n", BoldRed("[Sync Error]"), err)
				return
			}
			if granted {
				fmt.Printf("\n%s Telemetry sync consent recorded. Run %s to send usage.\n\n",
					BoldGreen("[Sync]"), BoldCyan("/sync"))
			} else {
				fmt.Printf("\n%s Telemetry sync consent revoked.\n\n", BoldGreen("[Sync]"))
			}
			return
		}

		if sub == "key" && len(parts) > 2 {
			key := strings.TrimSpace(parts[2])
			if key == "" {
				fmt.Printf("\n%s Sync key cannot be empty.\n\n", BoldRed("[Sync Error]"))
				return
			}
			s.Config.TelemetryKey = key
			s.Config.SetSetting("telemetry_key", key)
			if err := config.SaveConfig(s.Config); err != nil {
				fmt.Printf("\n%s Could not save sync key: %v\n\n", BoldRed("[Sync Error]"), err)
				return
			}
			prefix := key
			if len(prefix) > 16 {
				prefix = prefix[:16] + "..."
			}
			fmt.Printf("\n%s Telemetry API Sync Key set: %s\n\n", BoldGreen("[Sync]"), BoldCyan(prefix))
			return
		}

		if sub == "url" && len(parts) > 2 {
			endpoint := strings.TrimSpace(parts[2])
			if err := ValidateTelemetryEndpoint(endpoint); err != nil {
				fmt.Printf("\n%s Invalid telemetry URL: %v\n\n", BoldRed("[Sync Error]"), err)
				return
			}
			s.Config.TelemetryURL = endpoint
			s.Config.SetSetting("telemetry_url", endpoint)
			if err := config.SaveConfig(s.Config); err != nil {
				fmt.Printf("\n%s Could not save telemetry URL: %v\n\n", BoldRed("[Sync Error]"), err)
				return
			}
			fmt.Printf("\n%s Telemetry Sync URL set to: %s\n\n", BoldGreen("[Sync]"), BoldCyan(endpoint))
			return
		}
	}

	if !telemetrySyncConsented(s.Config) {
		fmt.Printf("\n%s Explicit consent is required before telemetry can be sent.\n", BoldYellow("[Sync Notice]"))
		fmt.Printf("  Run: %s\n\n", BoldCyan("/sync consent yes"))
		return
	}

	key := s.Config.GetTelemetryKey()
	if key == "" {
		fmt.Printf("\n%s Telemetry Sync Key not configured!\n", BoldYellow("[Sync Notice]"))
		fmt.Println("  1. Create an account on the mncode Web Dashboard.")
		fmt.Println("  2. Copy your API Key (`mnc_live_...`).")
		fmt.Printf("  3. Run: %s\n\n", BoldCyan("/sync key <your_key>"))
		return
	}

	endpoint := s.Config.GetTelemetryURL()
	fmt.Printf("\n%s Pushing session telemetry to cloud (%s)...\n", BoldCyan("[Sync]"), GrayText(endpoint))

	totalTokens, err := pushTelemetry(s, key, endpoint)
	if err != nil {
		fmt.Printf("%s %v\n\n", BoldRed("[Sync Error]"), err)
		return
	}

	today := time.Now().UTC().Format("2006-01-02")
	s.Config.SetSetting("last_telemetry_sync_date", today)
	if err := config.SaveConfig(s.Config); err != nil {
		fmt.Printf("%s Could not save last sync date: %v\n\n", BoldRed("[Sync Error]"), err)
		return
	}

	fmt.Printf("%s Telemetry pushed successfully! (Tokens: %s · Status: 200 OK)\n\n",
		BoldGreen("[Sync Success]"), BoldCyan(formatTokens(totalTokens)))
}

// buildTelemetryPayload derives every usage value from tracker records and the
// current session history. Unknown values are reported as zero/empty.
func buildTelemetryPayload(s *agent.Session, tracker *stats.Tracker, date string) TelemetryPayload {
	today := tracker.GetToday()
	records := tracker.Records()
	models := make(map[string]int)
	for _, record := range records {
		if record.Model != "" {
			models[record.Model]++
		}
	}

	toolsMap := make(map[string]int)
	if s != nil {
		for _, message := range s.History {
			for _, call := range message.ToolCalls {
				if call.Name != "" {
					toolsMap[call.Name]++
				}
			}
		}
	}

	return TelemetryPayload{
		ClientVersion:  config.CurrentVersion,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		SessionCount:   len(records),
		InputTokens:    int(today.InputTokens),
		OutputTokens:   int(today.OutputTokens),
		TotalTokens:    int(today.TotalTokens),
		ThinkingTokens: int(today.ThinkingTokens),
		Models:         models,
		Tools:          toolsMap,
		Date:           date,
	}
}

// ValidateTelemetryEndpoint requires telemetry to use HTTPS and rejects
// credentials in the URL. This applies to configured endpoints and redirects.
func ValidateTelemetryEndpoint(endpoint string) error {
	raw := strings.TrimSpace(endpoint)
	parsed, err := neturl.Parse(raw)
	if err != nil {
		return fmt.Errorf("malformed endpoint")
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("endpoint must use https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("endpoint host is required")
	}
	if parsed.User != nil {
		return fmt.Errorf("endpoint credentials are not allowed")
	}
	if parsed.Fragment != "" || strings.Contains(raw, "#") {
		return fmt.Errorf("endpoint fragments are not allowed")
	}
	return nil
}

func telemetrySyncConsented(c *config.Config) bool {
	if c == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(c.GetSetting("telemetry_sync_consent", ""))) {
	case "true", "yes", "1":
		return true
	default:
		return false
	}
}

// pushTelemetry builds today's usage payload and POSTs it to the telemetry
// endpoint. Returns the total token count pushed, for display by the caller.
func pushTelemetry(s *agent.Session, key, endpoint string) (int, error) {
	if s == nil || s.Config == nil {
		return 0, fmt.Errorf("session configuration unavailable")
	}
	if err := ValidateTelemetryEndpoint(endpoint); err != nil {
		return 0, err
	}

	tracker := stats.NewTracker()
	if err := tracker.LastError(); err != nil {
		return 0, fmt.Errorf("failed to load usage data: %w", err)
	}
	payload := buildTelemetryPayload(s, tracker, time.Now().UTC().Format("2006-01-02"))

	data, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to encode telemetry: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if err := ValidateTelemetryEndpoint(req.URL.String()); err != nil {
				return fmt.Errorf("unsafe telemetry redirect: %w", err)
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("connection failed: %w (make sure mncode-web is running at %s)", err, endpoint)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return 0, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return payload.TotalTokens, nil
}

// MaybeAutoSyncDaily performs an opt-in daily sync. The returned error lets
// callers surface network and persistence failures instead of swallowing them.
func MaybeAutoSyncDaily(s *agent.Session) error {
	if s == nil || s.Config == nil || !telemetrySyncConsented(s.Config) {
		return nil
	}
	key := s.Config.GetTelemetryKey()
	if key == "" {
		return nil
	}

	today := time.Now().UTC().Format("2006-01-02")
	if s.Config.GetSetting("last_telemetry_sync_date", "") == today {
		return nil
	}

	if _, err := pushTelemetry(s, key, s.Config.GetTelemetryURL()); err != nil {
		return err
	}

	s.Config.SetSetting("last_telemetry_sync_date", today)
	return config.SaveConfig(s.Config)
}
