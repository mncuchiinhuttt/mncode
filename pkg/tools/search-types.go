package tools

import (
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultBraveSearchEndpoint   = "https://api.search.brave.com/res/v1/web/search"
	defaultTavilySearchEndpoint  = "https://api.tavily.com/search"
	defaultGoogleSearchEndpoint  = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"
	defaultAntigravityEndpoint   = "https://daily-cloudcode-pa.googleapis.com/v1internal"
	defaultDuckDuckGoEndpoint    = "https://html.duckduckgo.com/html/"
	defaultAntigravityProjectURL = "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist"
	maxSearchResponseBytes       = 1 << 20
	maxSearchSnippetLength       = 2400
	maxSearchAnswerLength        = 6000
)

type searchResult struct {
	Title   string
	URL     string
	Snippet string
	Source  string
}

func searchHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func endpointOrDefault(endpoint, fallback string) string {
	if strings.TrimSpace(endpoint) == "" {
		return fallback
	}
	return strings.TrimRight(strings.TrimSpace(endpoint), "/")
}

func cleanURL(raw string) string {
	raw = html.UnescapeString(strings.TrimSpace(raw))
	if strings.Contains(raw, "uddg=") {
		if parsed, err := url.Parse(raw); err == nil {
			if actual := parsed.Query().Get("uddg"); actual != "" {
				return actual
			}
		}
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	return raw
}

var (
	searchHTMLTagPattern    = regexp.MustCompile(`<[^>]+>`)
	searchWhitespacePattern = regexp.MustCompile(`\s+`)
	duckResultPattern       = regexp.MustCompile(`(?s)<a[^>]+class="result__url"[^>]+href="([^"]+)"[^>]*>.*?<a[^>]+class="result__title"[^>]*>(.*?)</a>.*?<a[^>]+class="result__snippet"[^>]*>(.*?)</a>`)
	duckSimplePattern       = regexp.MustCompile(`(?s)<h2[^>]*class="result__title"[^>]*>\s*<a[^>]+href="([^"]+)"[^>]*>(.*?)</a>.*?<a[^>]+class="result__snippet"[^>]*>(.*?)</a>`)
)

func cleanSearchText(value string) string {
	value = searchHTMLTagPattern.ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	return strings.TrimSpace(searchWhitespacePattern.ReplaceAllString(value, " "))
}

func truncateSearchText(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "…"
}

func stripHTML(value string) string {
	return cleanSearchText(value)
}
