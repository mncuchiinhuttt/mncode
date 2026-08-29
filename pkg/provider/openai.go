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

type OpenAIProvider struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	accountID  string
}

func (o *OpenAIProvider) AccountID() string      { return o.accountID }
func (o *OpenAIProvider) SetAccountID(id string) { o.accountID = id }

func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (o *OpenAIProvider) Name() string { return "openai" }

func (o *OpenAIProvider) Stream(ctx context.Context, req *CompletionRequest, cb func(StreamEvent) error) (*CompletionResponse, error) {
	payload := o.buildPayload(req)
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	resp, err := o.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(o.Name(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes := readProviderErrorBody(resp.Body)
		return nil, newHTTPError(o.Name(), resp.StatusCode, string(bodyBytes), resp.Header)
	}
	return o.parseSSE(resp.Body, cb)
}

func (o *OpenAIProvider) buildPayload(req *CompletionRequest) map[string]interface{} {
	modelName := req.Model
	if modelName == "ox-alpha" {
		modelName = "stealth/ox-alpha"
	}

	payload := map[string]interface{}{
		"model":       modelName,
		"stream":      true,
		"temperature": req.Temperature,
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	if (strings.Contains(o.BaseURL, "openrouter") || strings.Contains(modelName, "ox-alpha")) && req.ThinkingBudget > 0 {
		payload["include_reasoning"] = true
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
