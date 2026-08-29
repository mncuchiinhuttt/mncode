package browserctl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RelayTab holds metadata for an open browser tab discovered over CDP.
type RelayTab struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	Type                 string `json:"type"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// DiscoverRelayTabs lists all inspectable tabs from the user's active Chrome browser.
func DiscoverRelayTabs(ctx context.Context, cdpEndpoint string) ([]RelayTab, error) {
	cdpEndpoint = strings.TrimSpace(cdpEndpoint)
	if cdpEndpoint == "" {
		cdpEndpoint = "127.0.0.1:9222"
	}
	if !strings.HasPrefix(cdpEndpoint, "http://") && !strings.HasPrefix(cdpEndpoint, "https://") {
		cdpEndpoint = "http://" + cdpEndpoint
	}

	reqURL := fmt.Sprintf("%s/json/list", strings.TrimSuffix(cdpEndpoint, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create cdp request: %w", err)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect to Chrome CDP at %s (ensure Chrome is launched with --remote-debugging-port=9222): %w", cdpEndpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CDP endpoint returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read CDP tabs: %w", err)
	}

	var allTabs []RelayTab
	if err := json.Unmarshal(data, &allTabs); err != nil {
		return nil, fmt.Errorf("parse CDP tab list: %w", err)
	}

	var pageTabs []RelayTab
	for _, t := range allTabs {
		if t.Type == "page" || t.Type == "" {
			pageTabs = append(pageTabs, t)
		}
	}

	return pageTabs, nil
}

// FindTargetTab matches an open tab by URL or title substring.
func FindTargetTab(tabs []RelayTab, target string) (*RelayTab, error) {
	target = strings.TrimSpace(target)
	if len(tabs) == 0 {
		return nil, fmt.Errorf("no open browser tabs found on relay")
	}
	if target == "" {
		// Default to first active visible page tab
		return &tabs[0], nil
	}

	targetLower := strings.ToLower(target)
	for _, t := range tabs {
		if strings.Contains(strings.ToLower(t.URL), targetLower) || strings.Contains(strings.ToLower(t.Title), targetLower) {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("no open tab matching %q found (discovered %d tabs)", target, len(tabs))
}
