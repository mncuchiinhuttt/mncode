package provider

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type OpenCodeProvider struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewOpenCodeProvider(apiKey, baseURL string) *OpenCodeProvider {
	if baseURL == "" {
		baseURL = "https://opencode.ai/zen/v1"
	}
	if apiKey == "" || apiKey == "public" {
		if envKey := os.Getenv("OPENCODE_API_KEY"); envKey != "" {
			apiKey = envKey
		}
	}
	return &OpenCodeProvider{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (o *OpenCodeProvider) Name() string { return "opencode" }

func (o *OpenCodeProvider) Stream(ctx context.Context, req *CompletionRequest, cb func(StreamEvent) error) (*CompletionResponse, error) {
	payload := o.buildPayload(req)
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal opencode request: %w", err)
	}

	url := o.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	sessionID := generateRandomHex(16)
	requestID := generateRandomHex(16)

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	httpReq.Header.Set("User-Agent", "opencode")
	httpReq.Header.Set("x-opencode-client", "desktop")
	httpReq.Header.Set("x-opencode-session", "ses_"+sessionID)
	httpReq.Header.Set("x-opencode-request", "msg_"+requestID)
	httpReq.Header.Set("x-opencode-project", "global")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := o.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(o.Name(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes := readProviderErrorBody(resp.Body)
		return nil, newHTTPError(o.Name(), resp.StatusCode, string(bodyBytes), resp.Header)
	}

	openAIProv := &OpenAIProvider{}
	return openAIProv.parseSSE(resp.Body, cb)
}

func (o *OpenCodeProvider) buildPayload(req *CompletionRequest) map[string]interface{} {
	payload := map[string]interface{}{
		"model":       req.Model,
		"stream":      true,
		"temperature": req.Temperature,
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}

	var msgs []map[string]interface{}
	if req.SystemPrompt != "" {
		msgs = append(msgs, map[string]interface{}{"role": "system", "content": req.SystemPrompt})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, formatOpenAIMessage(m))
	}
	payload["messages"] = msgs

	if len(req.Tools) > 0 {
		var tools []map[string]interface{}
		for _, t := range req.Tools {
			tools = append(tools, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.InputSchema,
				},
			})
		}
		payload["tools"] = tools
	}
	return payload
}

func generateRandomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
