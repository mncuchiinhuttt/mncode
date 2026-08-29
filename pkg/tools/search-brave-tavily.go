package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// fetchBraveSearch executes search via Brave Search API.
func fetchBraveSearch(ctx context.Context, query, apiKey string) ([]searchResult, error) {
	return fetchBraveSearchAt(ctx, query, apiKey, defaultBraveSearchEndpoint, nil)
}

func fetchBraveSearchAt(ctx context.Context, query, apiKey, endpoint string, client *http.Client) ([]searchResult, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("brave API key is missing")
	}
	u, err := url.Parse(endpointOrDefault(endpoint, defaultBraveSearchEndpoint))
	if err != nil {
		return nil, fmt.Errorf("invalid brave search endpoint: %w", err)
	}
	params := u.Query()
	params.Set("q", query)
	params.Set("count", "6")
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)
	resp, err := searchHTTPClient(client).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave search returned HTTP %d", resp.StatusCode)
	}

	var data struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxSearchResponseBytes)).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode brave response: %w", err)
	}

	results := make([]searchResult, 0, len(data.Web.Results))
	for _, item := range data.Web.Results {
		if strings.TrimSpace(item.URL) == "" || strings.TrimSpace(item.Title) == "" {
			continue
		}
		results = append(results, searchResult{
			Title:   truncateSearchText(cleanSearchText(item.Title), 500),
			URL:     item.URL,
			Snippet: truncateSearchText(cleanSearchText(item.Description), maxSearchSnippetLength),
			Source:  "Brave Search",
		})
	}
	return results, nil
}

// fetchTavilySearch executes search via Tavily Search API.
func fetchTavilySearch(ctx context.Context, query, apiKey string) ([]searchResult, error) {
	return fetchTavilySearchAt(ctx, query, apiKey, defaultTavilySearchEndpoint, nil)
}

func fetchTavilySearchAt(ctx context.Context, query, apiKey, endpoint string, client *http.Client) ([]searchResult, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("tavily API key is missing")
	}
	payload := map[string]interface{}{
		"api_key":        apiKey,
		"query":          query,
		"search_depth":   "basic",
		"include_answer": true,
		"max_results":    6,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode tavily request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointOrDefault(endpoint, defaultTavilySearchEndpoint), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := searchHTTPClient(client).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily search returned HTTP %d", resp.StatusCode)
	}

	var data struct {
		Answer  string `json:"answer"`
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxSearchResponseBytes)).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode tavily response: %w", err)
	}

	results := make([]searchResult, 0, len(data.Results)+1)
	if answer := cleanSearchText(data.Answer); answer != "" {
		results = append(results, searchResult{Title: "Tavily answer", Snippet: truncateSearchText(answer, maxSearchAnswerLength), Source: "Tavily AI"})
	}
	for _, item := range data.Results {
		if strings.TrimSpace(item.URL) == "" || strings.TrimSpace(item.Title) == "" {
			continue
		}
		results = append(results, searchResult{
			Title:   truncateSearchText(cleanSearchText(item.Title), 500),
			URL:     item.URL,
			Snippet: truncateSearchText(cleanSearchText(item.Content), maxSearchSnippetLength),
			Source:  "Tavily Search",
		})
	}
	return results, nil
}
