package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type AnthropicProvider struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewAnthropicProvider(apiKey, baseURL string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	return &AnthropicProvider{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (a *AnthropicProvider) Name() string { return "anthropic" }

func (a *AnthropicProvider) Stream(ctx context.Context, req *CompletionRequest, cb func(StreamEvent) error) (*CompletionResponse, error) {
	payload := a.buildPayload(req)
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("anthropic-beta", "output-128k-2025-02-19")

	resp, err := a.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(a.Name(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes := readProviderErrorBody(resp.Body)
		return nil, newHTTPError(a.Name(), resp.StatusCode, string(bodyBytes), resp.Header)
	}

	return a.parseSSE(resp.Body, cb)
}

func (a *AnthropicProvider) buildPayload(req *CompletionRequest) map[string]interface{} {
	payload := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"stream":     true,
	}
	if req.SystemPrompt != "" {
		payload["system"] = req.SystemPrompt
	}
	if req.ThinkingBudget > 0 && strings.Contains(req.Model, "3-7") {
		payload["thinking"] = map[string]interface{}{
			"type":          "enabled",
			"budget_tokens": req.ThinkingBudget,
		}
	} else if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}

	var msgs []map[string]interface{}
	for _, m := range req.Messages {
		msgs = append(msgs, formatAnthropicMessage(m))
	}
	payload["messages"] = msgs

	if len(req.Tools) > 0 {
		var tools []map[string]interface{}
		for _, t := range req.Tools {
			tools = append(tools, map[string]interface{}{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": t.InputSchema,
			})
		}
		payload["tools"] = tools
	}
	return payload
}
