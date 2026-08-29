package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// fetchAntigravitySearch executes Google Search Grounding through Antigravity OAuth or Gemini API key.
func fetchAntigravitySearch(ctx context.Context, query, token, projectID, geminiKey string) ([]searchResult, error) {
	return fetchAntigravitySearchAt(ctx, query, token, projectID, geminiKey, defaultAntigravityEndpoint, defaultGoogleSearchEndpoint, nil)
}

func fetchAntigravitySearchAt(ctx context.Context, query, token, projectID, geminiKey, cloudEndpoint, geminiEndpoint string, client *http.Client) ([]searchResult, error) {
	if strings.TrimSpace(token) == "" && strings.TrimSpace(geminiKey) == "" {
		return nil, fmt.Errorf("no Google OAuth access token or Gemini API key available")
	}
	client = searchHTTPClient(client)

	if strings.TrimSpace(token) != "" {
		token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
		if strings.TrimSpace(projectID) == "" && endpointOrDefault(cloudEndpoint, defaultAntigravityEndpoint) == defaultAntigravityEndpoint {
			projectID = loadAntigravityProjectID(ctx, token, client)
		}
		request := map[string]interface{}{
			"contents": []map[string]interface{}{{
				"role":  "user",
				"parts": []map[string]interface{}{{"text": query}},
			}},
			"tools": []map[string]interface{}{{"googleSearch": map[string]interface{}{}}},
		}
		envelope := map[string]interface{}{
			"model":       "gemini-2.5-flash",
			"request":     request,
			"userAgent":   "antigravity",
			"requestType": "agent",
			"requestId":   fmt.Sprintf("mncode-search-%d", time.Now().UnixNano()),
		}
		if strings.TrimSpace(projectID) != "" {
			envelope["project"] = projectID
		}
		body, err := json.Marshal(envelope)
		if err != nil {
			return nil, fmt.Errorf("failed to encode Antigravity request: %w", err)
		}
		target := endpointOrDefault(cloudEndpoint, defaultAntigravityEndpoint) + ":streamGenerateContent?alt=sse"
		return doGroundedRequest(ctx, target, body, token, client, true)
	}

	body, err := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{{
			"role":  "user",
			"parts": []map[string]interface{}{{"text": query}},
		}},
		"tools": []map[string]interface{}{{"googleSearch": map[string]interface{}{}}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode Gemini request: %w", err)
	}
	return doGroundedRequest(ctx, endpointOrDefault(geminiEndpoint, defaultGoogleSearchEndpoint), body, geminiKey, client, false)
}

func loadAntigravityProjectID(ctx context.Context, token string, client *http.Client) string {
	body := []byte(`{"metadata":{"ideType":9,"platform":2,"pluginType":2},"mode":1}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, defaultAntigravityProjectURL, bytes.NewReader(body))
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1 vscode-antigravity/1.107.0")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var data struct {
		ProjectID string `json:"cloudaicompanionProject"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 32*1024)).Decode(&data) != nil {
		return ""
	}
	return strings.TrimSpace(data.ProjectID)
}

func doGroundedRequest(ctx context.Context, target string, body []byte, credential string, client *http.Client, antigravity bool) ([]searchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	if antigravity {
		req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(credential, "Bearer "))
		req.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1 vscode-antigravity/1.107.0")
		req.Header.Set("X-Client-Name", "antigravity")
		req.Header.Set("X-Client-Version", "1.107.0")
		req.Header.Set("X-Goog-Api-Client", "antigravity/1.107.0")
	} else {
		req.Header.Set("x-goog-api-key", credential)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google Search Grounding returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSearchResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read Google Search Grounding response: %w", err)
	}
	results := parseGroundingResponse(data)
	if len(results) == 0 {
		return nil, fmt.Errorf("Google Search Grounding returned no sources")
	}
	return results, nil
}
