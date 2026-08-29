package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// fetchDuckDuckGoSearch executes search via DuckDuckGo HTML scraping.
func fetchDuckDuckGoSearch(ctx context.Context, query string) ([]searchResult, error) {
	return fetchDuckDuckGoSearchAt(ctx, query, defaultDuckDuckGoEndpoint, nil)
}

func fetchDuckDuckGoSearchAt(ctx context.Context, query, endpoint string, client *http.Client) ([]searchResult, error) {
	u, err := url.Parse(endpointOrDefault(endpoint, defaultDuckDuckGoEndpoint))
	if err != nil {
		return nil, fmt.Errorf("invalid DuckDuckGo endpoint: %w", err)
	}
	params := u.Query()
	params.Set("q", query)
	u.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := searchHTTPClient(client).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("DuckDuckGo returned HTTP %d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, err
	}
	body := string(bodyBytes)
	results := make([]searchResult, 0, 10)
	for _, match := range duckResultPattern.FindAllStringSubmatch(body, 10) {
		if rawURL := cleanURL(match[1]); rawURL != "" {
			results = append(results, searchResult{
				Title:   truncateSearchText(cleanSearchText(match[2]), 500),
				URL:     rawURL,
				Snippet: truncateSearchText(cleanSearchText(match[3]), maxSearchSnippetLength),
				Source:  "DuckDuckGo",
			})
		}
	}
	if len(results) == 0 {
		for _, match := range duckSimplePattern.FindAllStringSubmatch(body, 10) {
			if rawURL := cleanURL(match[1]); rawURL != "" {
				results = append(results, searchResult{
					Title:   truncateSearchText(cleanSearchText(match[2]), 500),
					URL:     rawURL,
					Snippet: truncateSearchText(cleanSearchText(match[3]), maxSearchSnippetLength),
					Source:  "DuckDuckGo",
				})
			}
		}
	}
	if len(results) == 0 && strings.Contains(strings.ToLower(body), "blocked") {
		return nil, fmt.Errorf("DuckDuckGo rate limit or bot block encountered")
	}
	return results, nil
}
