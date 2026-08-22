package provider

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

type GeminiProvider struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewGeminiProvider(apiKey, baseURL string) *GeminiProvider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	return &GeminiProvider{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (g *GeminiProvider) Name() string { return "gemini" }

func (g *GeminiProvider) Stream(ctx context.Context, req *CompletionRequest, cb func(StreamEvent) error) (*CompletionResponse, error) {
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", g.BaseURL, req.Model)
	isOAuth := strings.HasPrefix(g.APIKey, "ya29.") || strings.HasPrefix(g.APIKey, "Bearer ")
	if !isOAuth && g.APIKey != "" {
		url = fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", g.BaseURL, req.Model, g.APIKey)
	}

	payload := g.buildPayload(req)
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if isOAuth {
		tok := g.APIKey
		if !strings.HasPrefix(tok, "Bearer ") {
			tok = "Bearer " + tok
		}
		httpReq.Header.Set("Authorization", tok)
	} else if g.APIKey != "" {
		httpReq.Header.Set("x-goog-api-key", g.APIKey)
	}

	resp, err := g.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("gemini API error (status %d): %s\n\n\033[1;33m[Tip]\033[0m Run '/login antigravity' or '/account import' or configure your API key with '/config'.", resp.StatusCode, string(bodyBytes))
		}
		return nil, fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return g.parseSSE(resp.Body, cb)
}

func (g *GeminiProvider) buildPayload(req *CompletionRequest) map[string]interface{} {
	payload := map[string]interface{}{}

	if req.SystemPrompt != "" {
		payload["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{{"text": req.SystemPrompt}},
		}
	}

	var contents []map[string]interface{}
	for _, m := range req.Messages {
		role := "user"
		if m.Role == RoleAssistant {
			role = "model"
		}
		var parts []map[string]interface{}
		for _, img := range m.Images {
			parts = append(parts, map[string]interface{}{
				"inlineData": map[string]interface{}{
					"mimeType": img.MediaType,
					"data":     img.Data,
				},
			})
		}
		if m.Content != "" {
			parts = append(parts, map[string]interface{}{"text": m.Content})
		}
		for _, tc := range m.ToolCalls {
			part := map[string]interface{}{
				"functionCall": map[string]interface{}{
					"name": tc.Name,
					"args": tc.Arguments,
				},
			}
			if tc.ThoughtSignature != "" {
				part["thoughtSignature"] = tc.ThoughtSignature
			}
			parts = append(parts, part)
		}
		for _, tr := range m.ToolResults {
			parts = append(parts, map[string]interface{}{
				"functionResponse": map[string]interface{}{
					"name": tr.Name,
					"response": map[string]interface{}{"content": tr.Content},
				},
			})
		}
		contents = append(contents, map[string]interface{}{"role": role, "parts": parts})
	}
	payload["contents"] = contents

	if len(req.Tools) > 0 {
		var decls []map[string]interface{}
		for _, t := range req.Tools {
			decls = append(decls, map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			})
		}
		payload["tools"] = []map[string]interface{}{{"functionDeclarations": decls}}
	}

	return payload
}
