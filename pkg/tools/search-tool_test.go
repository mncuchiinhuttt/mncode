package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"mncode/pkg/config"
)

type searchRoundTripper func(*http.Request) (*http.Response, error)

func (f searchRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func searchResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d", status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestSearchWebToolSchemaAndEmptyQuery(t *testing.T) {
	tool := &SearchWebTool{}
	if tool.Name() != "search_web" {
		t.Fatalf("Name() = %q, want search_web", tool.Name())
	}
	if schema := tool.Schema(); schema["type"] != "object" {
		t.Fatalf("schema type = %v, want object", schema["type"])
	}
	if _, err := tool.Execute(context.Background(), map[string]interface{}{}); err == nil {
		t.Fatal("Execute() accepted an empty query")
	}
}

func TestFetchBraveSearchRequestAndResponse(t *testing.T) {
	var got *http.Request
	client := &http.Client{Transport: searchRoundTripper(func(req *http.Request) (*http.Response, error) {
		got = req
		return searchResponse(http.StatusOK, `{"web":{"results":[{"title":"Go docs","url":"https://go.dev/doc/","description":"Official docs"}]}}`), nil
	})}

	results, err := fetchBraveSearchAt(context.Background(), "golang docs", "brave-secret", "", client)
	if err != nil {
		t.Fatalf("fetchBraveSearchAt() error = %v", err)
	}
	if len(results) != 1 || results[0].URL != "https://go.dev/doc/" {
		t.Fatalf("unexpected Brave results: %#v", results)
	}
	if got.Header.Get("X-Subscription-Token") != "brave-secret" {
		t.Fatalf("Brave token header = %q", got.Header.Get("X-Subscription-Token"))
	}
	if got.URL.Query().Get("q") != "golang docs" || got.URL.Query().Get("count") != "6" {
		t.Fatalf("unexpected Brave query: %s", got.URL.String())
	}
}

func TestFetchTavilySearchRequestAndResponse(t *testing.T) {
	var gotPayload map[string]interface{}
	client := &http.Client{Transport: searchRoundTripper(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(body, &gotPayload)
		return searchResponse(http.StatusOK, `{"answer":"Use the official docs.","results":[{"title":"Go","url":"https://go.dev/","content":"The Go homepage."}]}`), nil
	})}

	results, err := fetchTavilySearchAt(context.Background(), "golang", "tavily-secret", "", client)
	if err != nil {
		t.Fatalf("fetchTavilySearchAt() error = %v", err)
	}
	if gotPayload["api_key"] != "tavily-secret" || gotPayload["query"] != "golang" {
		t.Fatalf("unexpected Tavily payload: %#v", gotPayload)
	}
	if len(results) != 2 || results[0].Source != "Tavily AI" || results[1].URL != "https://go.dev/" {
		t.Fatalf("unexpected Tavily results: %#v", results)
	}
}

func TestFetchAntigravitySearchUsesCloudCodeEnvelope(t *testing.T) {
	var got *http.Request
	client := &http.Client{Transport: searchRoundTripper(func(req *http.Request) (*http.Response, error) {
		got = req
		return searchResponse(http.StatusOK, "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Grounded answer\"}]},\"groundingMetadata\":{\"groundingChunks\":[{\"web\":{\"uri\":\"https://example.com/source\",\"title\":\"Example source\"}}]}}]}}\n\ndata: [DONE]\n"), nil
	})}

	results, err := fetchAntigravitySearchAt(context.Background(), "latest Go release", "ya29-oauth", "project-123", "", "", "", client)
	if err != nil {
		t.Fatalf("fetchAntigravitySearchAt() error = %v", err)
	}
	if !strings.HasSuffix(got.URL.Path, ":streamGenerateContent") || got.URL.Query().Get("alt") != "sse" {
		t.Fatalf("unexpected Cloud Code URL: %s", got.URL.String())
	}
	if got.Header.Get("Authorization") != "Bearer ya29-oauth" || got.Header.Get("X-Client-Name") != "antigravity" {
		t.Fatalf("unexpected Antigravity headers: %#v", got.Header)
	}
	var envelope map[string]interface{}
	body, _ := io.ReadAll(got.Body)
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope["project"] != "project-123" || envelope["userAgent"] != "antigravity" {
		t.Fatalf("unexpected Cloud Code envelope: %#v", envelope)
	}
	request, ok := envelope["request"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing nested request: %#v", envelope)
	}
	tools, ok := request["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Fatalf("missing googleSearch tool: %#v", request["tools"])
	}
	if len(results) != 2 || results[0].Snippet != "Grounded answer" || results[1].URL != "https://example.com/source" {
		t.Fatalf("unexpected grounded results: %#v", results)
	}
}

func TestFetchGeminiSearchUsesHeaderKeyAndDirectResponse(t *testing.T) {
	var got *http.Request
	client := &http.Client{Transport: searchRoundTripper(func(req *http.Request) (*http.Response, error) {
		got = req
		return searchResponse(http.StatusOK, `{"candidates":[{"content":{"parts":[{"text":"Gemini answer"}]},"groundingMetadata":{"groundingChunks":[{"web":{"uri":"https://example.com/gemini","title":"Gemini source"}}]}}]}`), nil
	})}

	results, err := fetchAntigravitySearchAt(context.Background(), "query", "", "", "gemini-secret", "", "", client)
	if err != nil {
		t.Fatalf("fetchAntigravitySearchAt() Gemini error = %v", err)
	}
	if got.Header.Get("x-goog-api-key") != "gemini-secret" || got.URL.Query().Get("key") != "" {
		t.Fatalf("Gemini credential leaked or missing header: URL=%s headers=%v", got.URL, got.Header)
	}
	if len(results) != 2 || results[1].URL != "https://example.com/gemini" {
		t.Fatalf("unexpected Gemini results: %#v", results)
	}
}

func TestSearchWebToolFallbacksAfterSelectedEngineFailure(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SearchEngine = "brave"
	cfg.BraveAPIKey = "brave-secret"
	client := &http.Client{Transport: searchRoundTripper(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Host, "brave") {
			return searchResponse(http.StatusUnauthorized, `{}`), nil
		}
		return searchResponse(http.StatusOK, `<a class="result__url" href="https://example.com">example.com</a><a class="result__title">Fallback result</a><a class="result__snippet">DuckDuckGo snippet</a>`), nil
	})}
	tool := &SearchWebTool{Config: cfg, HTTPClient: client}

	results, engine, err := tool.searchWithFallback(context.Background(), "fallback query")
	if err != nil || engine != "DuckDuckGo" || len(results) != 1 {
		t.Fatalf("fallback = %#v, %q, %v", results, engine, err)
	}
}

func TestSearchHelpers(t *testing.T) {
	if got := cleanURL("https://duckduckgo.com/l/?uddg=https%3A%2F%2Fgo.dev%2F"); got != "https://go.dev/" {
		t.Fatalf("cleanURL() = %q", got)
	}
	if got := stripHTML("<b>Go</b>  docs"); got != "Go docs" {
		t.Fatalf("stripHTML() = %q", got)
	}
}

func TestTruncateSearchTextIsRuneSafe(t *testing.T) {
	if got := truncateSearchText("こんにちは世界", 5); got != "こんにちは…" {
		t.Fatalf("truncateSearchText() = %q", got)
	}
	if got := truncateSearchText("short", 10); got != "short" {
		t.Fatalf("truncateSearchText() changed short text: %q", got)
	}
}
