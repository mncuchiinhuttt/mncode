package tools

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mncode/pkg/accounts"
	"mncode/pkg/config"
)

// SearchWebTool performs real-time web search with multi-engine support (Brave, Tavily, Google Antigravity, DuckDuckGo)
type SearchWebTool struct {
	Config             *config.Config
	Accounts           *accounts.Store
	GoogleTokenGetter  func() string
	ProjectIDGetter    func() string
	HTTPClient         *http.Client
	BraveEndpoint      string
	TavilyEndpoint     string
	GoogleEndpoint     string
	DuckDuckGoEndpoint string
}

const maxRenderedSearchResults = 10

func (s *SearchWebTool) Name() string {
	return "search_web"
}

func (s *SearchWebTool) Description() string {
	return "Agent-invoked live web search for technical documentation, library APIs, error solutions, GitHub issues, and package references. The harness calls this tool automatically when web research is needed; users do not need a /search command to search. Uses the configured engine (Brave, Tavily, Google Grounding, DuckDuckGo) with automatic fallback."
}

func (s *SearchWebTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query terms (e.g. 'Golang sync.Map example', 'Next.js 15 metadata', 'how to fix ECONNREFUSED')",
			},
			"domain": map[string]interface{}{
				"type":        "string",
				"description": "Optional domain filter (e.g. 'github.com', 'pkg.go.dev', 'stackoverflow.com')",
			},
		},
		"required": []string{"query"},
	}
}

func (s *SearchWebTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, _ := args["query"].(string)
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("query is required")
	}

	domain, _ := args["domain"].(string)
	if strings.TrimSpace(domain) != "" {
		query = fmt.Sprintf("site:%s %s", strings.TrimSpace(domain), query)
	}
	results, engineUsed, err := s.searchWithFallback(ctx, query)
	if err != nil || len(results) == 0 {
		return fmt.Sprintf("No live web search results found for query: '%s'. You may also fetch URLs directly using read_url_content.", query), nil
	}
	if len(results) > maxRenderedSearchResults {
		results = results[:maxRenderedSearchResults]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Web Search Results for '%s' (via %s):\n\n", query, engineUsed))
	for i, result := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		if result.URL != "" {
			sb.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		}
		if result.Source != "" {
			sb.WriteString(fmt.Sprintf("   Source: %s\n", result.Source))
		}
		if result.Snippet != "" {
			sb.WriteString(fmt.Sprintf("   Snippet: %s\n", result.Snippet))
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func (s *SearchWebTool) searchWithFallback(ctx context.Context, query string) ([]searchResult, string, error) {
	engine := "auto"
	if s.Config != nil {
		engine = s.Config.GetSearchEngine()
	}

	braveKey, tavilyKey := "", ""
	googleToken, projectID, geminiKey := "", "", ""
	if s.Config != nil {
		braveKey = s.Config.GetBraveAPIKey()
		tavilyKey = s.Config.GetTavilyAPIKey()
		switch s.Config.Provider {
		case config.ProviderAntigravity:
			googleToken = s.Config.APIKey
		case config.ProviderGemini:
			geminiKey = s.Config.APIKey
		}
		if strings.HasPrefix(strings.TrimSpace(s.Config.APIKey), "ya29.") {
			googleToken = s.Config.APIKey
		}
	}
	if (engine == "auto" || engine == "antigravity") && s.Accounts != nil {
		if stored := s.Accounts.GetActiveAccount(accounts.ProviderTypeAntigravity); stored != nil {
			account := *stored
			if account.IsAvailable() {
				if account.RefreshToken != "" &&
					(account.AccessToken == "" || account.ExpiresAt.IsZero() || (!account.ExpiresAt.IsZero() && time.Now().Add(30*time.Second).After(account.ExpiresAt))) {
					if refreshed, err := accounts.RefreshGoogleToken(account.RefreshToken, "", ""); err == nil && refreshed != "" {
						account.AccessToken = refreshed
						account.ExpiresAt = time.Now().Add(time.Hour)
						_ = s.Accounts.AddOrUpdate(&account)
					}
				}
				if account.AccessToken != "" {
					googleToken = account.AccessToken
				}
			}
		}
	}
	if engine == "auto" || engine == "antigravity" {
		if s.GoogleTokenGetter != nil {
			if token := strings.TrimSpace(s.GoogleTokenGetter()); token != "" {
				googleToken = token
			}
		}
		if s.ProjectIDGetter != nil {
			projectID = strings.TrimSpace(s.ProjectIDGetter())
		}
	}

	order := []string{}
	if engine != "auto" {
		order = append(order, engine)
	}
	order = append(order, "antigravity", "tavily", "brave", "duckduckgo")
	seen := make(map[string]struct{}, len(order))
	var lastErr error
	for _, candidate := range order {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		var results []searchResult
		var err error
		switch candidate {
		case "antigravity":
			if googleToken == "" && geminiKey == "" {
				continue
			}
			results, err = fetchAntigravitySearchAt(ctx, query, googleToken, projectID, geminiKey, s.GoogleEndpoint, s.GoogleEndpoint, s.HTTPClient)
		case "tavily":
			if tavilyKey == "" {
				continue
			}
			results, err = fetchTavilySearchAt(ctx, query, tavilyKey, s.TavilyEndpoint, s.HTTPClient)
		case "brave":
			if braveKey == "" {
				continue
			}
			results, err = fetchBraveSearchAt(ctx, query, braveKey, s.BraveEndpoint, s.HTTPClient)
		case "duckduckgo":
			results, err = fetchDuckDuckGoSearchAt(ctx, query, s.DuckDuckGoEndpoint, s.HTTPClient)
		}
		if err != nil {
			lastErr = err
			continue
		}
		if len(results) > 0 {
			return results, searchEngineLabel(candidate), nil
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no configured search backend returned results")
	}
	return nil, "None", lastErr
}

func searchEngineLabel(engine string) string {
	switch engine {
	case "antigravity":
		return "Google Search Grounding"
	case "tavily":
		return "Tavily Search"
	case "brave":
		return "Brave Search"
	default:
		return "DuckDuckGo"
	}
}
