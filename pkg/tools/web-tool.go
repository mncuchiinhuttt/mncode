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

// WebTool fetches web page content and converts it to clean markdown
type WebTool struct{}

func (w *WebTool) Name() string {
	return "read_url_content"
}

func (w *WebTool) Description() string {
	return "Fetch textual documentation and clean content from a public web URL via HTTP request."
}

func (w *WebTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch content from (e.g. 'https://docs.github.com/en', 'https://pkg.go.dev/net/http').",
			},
		},
		"required": []string{"Url"},
	}
}

func (w *WebTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	urlStr, _ := args["Url"].(string)
	if urlStr == "" {
		urlStr, _ = args["url"].(string)
	}
	if strings.TrimSpace(urlStr) == "" {
		return "", fmt.Errorf("Url is required")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain")

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

	text := htmlToMarkdown(string(body))
	if len(text) > 12000 {
		text = text[:12000] + "\n\n...[Content truncated to 12,000 characters]"
	}

	return fmt.Sprintf("URL: %s\nStatus: %d\n\n%s", urlStr, resp.StatusCode, text), nil
}

func htmlToMarkdown(html string) string {
	// Strip script and style
	reScript := regexp.MustCompile(`(?is)<(script|style|svg|noscript|iframe)[^>]*>.*?</\1>`)
	cleaned := reScript.ReplaceAllString(html, "")

	// Preserve pre/code blocks
	reCode := regexp.MustCompile(`(?is)<pre[^>]*><code[^>]*>(.*?)</code></pre>`)
	cleaned = reCode.ReplaceAllString(cleaned, "\n```\n$1\n```\n")

	// Headings
	reH1 := regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	cleaned = reH1.ReplaceAllString(cleaned, "\n# $1\n")
	reH2 := regexp.MustCompile(`(?is)<h2[^>]*>(.*?)</h2>`)
	cleaned = reH2.ReplaceAllString(cleaned, "\n## $1\n")
	reH3 := regexp.MustCompile(`(?is)<h3[^>]*>(.*?)</h3>`)
	cleaned = reH3.ReplaceAllString(cleaned, "\n### $1\n")

	// Links
	reA := regexp.MustCompile(`(?is)<a[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)
	cleaned = reA.ReplaceAllString(cleaned, "[$2]($1)")

	// Paragraphs & Line breaks
	reP := regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	cleaned = reP.ReplaceAllString(cleaned, "\n$1\n")
	reBr := regexp.MustCompile(`(?i)<br\s*/?>`)
	cleaned = reBr.ReplaceAllString(cleaned, "\n")
	reLi := regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	cleaned = reLi.ReplaceAllString(cleaned, "\n- $1")

	// Strip remaining HTML tags
	reTags := regexp.MustCompile(`<[^>]+>`)
	cleaned = reTags.ReplaceAllString(cleaned, " ")

	// Normalize spaces & blank lines
	reSpaces := regexp.MustCompile(`[ \t]+`)
	cleaned = reSpaces.ReplaceAllString(cleaned, " ")
	reBlankLines := regexp.MustCompile(`\n{3,}`)
	cleaned = reBlankLines.ReplaceAllString(cleaned, "\n\n")

	return strings.TrimSpace(cleaned)
}
