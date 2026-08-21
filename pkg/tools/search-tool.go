package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// SearchWebTool performs real-time web search for docs, solutions & error research
type SearchWebTool struct{}

func (s *SearchWebTool) Name() string {
	return "search_web"
}

func (s *SearchWebTool) Description() string {
	return "Performs a live web search for technical documentation, library APIs, error solutions, GitHub issues, and package references."
}

func (s *SearchWebTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query keywords (e.g. 'golang fiber middleware jwt', 'react 19 useActionState example').",
			},
			"domain": map[string]interface{}{
				"type":        "string",
				"description": "Optional domain filter (e.g. 'github.com', 'pkg.go.dev', 'stackoverflow.com').",
			},
		},
		"required": []string{"query"},
	}
}

type searchResult struct {
	Title   string
	URL     string
	Snippet string
}

func (s *SearchWebTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, _ := args["query"].(string)
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("query is required")
	}

	domain, _ := args["domain"].(string)
	if domain != "" {
		query = fmt.Sprintf("site:%s %s", domain, query)
	}

	results, err := fetchDuckDuckGoSearch(ctx, query)
	if err != nil || len(results) == 0 {
		return fmt.Sprintf("No live web search results found for query: '%s'. You may also fetch URLs directly using read_url_content.", query), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Web Search Results for '%s':\n\n", query))
	for i, r := range results {
		if i >= 6 {
			break
		}
		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("- **URL:** %s\n", r.URL))
		sb.WriteString(fmt.Sprintf("- **Snippet:** %s\n\n", r.Snippet))
	}

	return sb.String(), nil
}

func fetchDuckDuckGoSearch(ctx context.Context, query string) ([]searchResult, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, err
	}
	body := string(bodyBytes)

	var results []searchResult

	// Regex to extract title, url, snippet from DuckDuckGo HTML format
	reResult := regexp.MustCompile(`(?s)<a[^>]+class="result__url"[^>]+href="([^"]+)"[^>]*>.*?<a[^>]+class="result__title"[^>]*>(.*?)</a>.*?<a[^>]+class="result__snippet"[^>]*>(.*?)</a>`)
	matches := reResult.FindAllStringSubmatch(body, 10)

	for _, m := range matches {
		rawURL := cleanURL(m[1])
		title := stripHTML(m[2])
		snippet := stripHTML(m[3])

		if rawURL != "" && title != "" {
			results = append(results, searchResult{
				Title:   title,
				URL:     rawURL,
				Snippet: snippet,
			})
		}
	}

	// Fallback simpler regex if DDG altered wrapper class names
	if len(results) == 0 {
		reSimple := regexp.MustCompile(`(?s)<h2[^>]*class="result__title"[^>]*>\s*<a[^>]+href="([^"]+)"[^>]*>(.*?)</a>.*?<a[^>]+class="result__snippet"[^>]*>(.*?)</a>`)
		matchesSimple := reSimple.FindAllStringSubmatch(body, 10)
		for _, m := range matchesSimple {
			results = append(results, searchResult{
				Title:   stripHTML(m[2]),
				URL:     cleanURL(m[1]),
				Snippet: stripHTML(m[3]),
			})
		}
	}

	return results, nil
}

func cleanURL(raw string) string {
	if strings.Contains(raw, "uddg=") {
		u, err := url.Parse(raw)
		if err == nil {
			if actual := u.Query().Get("uddg"); actual != "" {
				return actual
			}
		}
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	return raw
}

func stripHTML(html string) string {
	reTags := regexp.MustCompile(`<[^>]+>`)
	cleaned := reTags.ReplaceAllString(html, " ")
	reSpaces := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(reSpaces.ReplaceAllString(cleaned, " "))
}
