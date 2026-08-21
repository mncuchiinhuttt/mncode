package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// WebTool fetches web page content
type WebTool struct{}

func (w *WebTool) Name() string {
	return "read_url_content"
}

func (w *WebTool) Description() string {
	return "Fetch textual content from a URL via HTTP request."
}

func (w *WebTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch content from.",
			},
		},
		"required": []string{"Url"},
	}
}

func (w *WebTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	urlStr, _ := args["Url"].(string)
	if urlStr == "" {
		return "", fmt.Errorf("Url is required")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "mncode-cli/1.0")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP error %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Basic HTML stripping
	text := stripHTML(string(body))
	if len(text) > 8000 {
		text = text[:8000] + "\n\n...[Content truncated]"
	}

	return fmt.Sprintf("URL: %s\nStatus: %d\n\n%s", urlStr, resp.StatusCode, text), nil
}

func stripHTML(html string) string {
	reScript := regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</\1>`)
	cleaned := reScript.ReplaceAllString(html, "")

	reTags := regexp.MustCompile(`<[^>]+>`)
	cleaned = reTags.ReplaceAllString(cleaned, " ")

	reSpaces := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(reSpaces.ReplaceAllString(cleaned, " "))
}
