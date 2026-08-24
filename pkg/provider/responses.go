package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ResponsesProvider handles the OpenAI Responses API format for custom providers.
// Tool-call streaming is intentionally left to the provider's native response
// events; text and reasoning deltas remain fully compatible with the agent loop.
type ResponsesProvider struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewResponsesProvider(apiKey, baseURL string) *ResponsesProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &ResponsesProvider{APIKey: apiKey, BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: &http.Client{Timeout: 10 * time.Minute}}
}

func (r *ResponsesProvider) Name() string { return "responses" }

func (r *ResponsesProvider) Stream(ctx context.Context, req *CompletionRequest, cb func(StreamEvent) error) (*CompletionResponse, error) {
	payload := map[string]interface{}{"model": req.Model, "stream": true, "input": responsesInput(req)}
	if req.MaxTokens > 0 {
		payload["max_output_tokens"] = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, map[string]interface{}{"type": "function", "name": tool.Name, "description": tool.Description, "parameters": tool.InputSchema, "strict": false})
		}
		payload["tools"] = tools
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal responses request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.BaseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+r.APIKey)
	response, err := r.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("responses API request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("responses API error (status %d): %s", response.StatusCode, string(bodyBytes))
	}
	return parseResponsesSSE(response.Body, cb)
}

func responsesInput(req *CompletionRequest) []map[string]interface{} {
	input := make([]map[string]interface{}, 0, len(req.Messages)+1)
	if req.SystemPrompt != "" {
		input = append(input, map[string]interface{}{"role": "developer", "content": []map[string]string{{"type": "input_text", "text": req.SystemPrompt}}})
	}
	for _, message := range req.Messages {
		input = append(input, map[string]interface{}{"role": string(message.Role), "content": []map[string]string{{"type": "input_text", "text": message.Content}}})
	}
	return input
}

func parseResponsesSSE(reader io.Reader, cb func(StreamEvent) error) (*CompletionResponse, error) {
	scanner := bufio.NewScanner(reader)
	result := &CompletionResponse{}
	eventName := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventName = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &data); err != nil {
			continue
		}
		delta, _ := data["delta"].(string)
		if strings.Contains(eventName, "output_text") && delta != "" {
			result.Content += delta
			_ = cb(StreamEvent{Type: EventToken, Text: delta})
		}
		if strings.Contains(eventName, "reasoning") && delta != "" {
			result.Thinking += delta
			_ = cb(StreamEvent{Type: EventThinking, Thinking: delta})
		}
		if usage, ok := data["usage"].(map[string]interface{}); ok {
			if value, ok := usage["input_tokens"].(float64); ok {
				result.InputTokens = int(value)
			}
			if value, ok := usage["output_tokens"].(float64); ok {
				result.OutputTokens = int(value)
			}
		}
	}
	_ = cb(StreamEvent{Type: EventDone})
	return result, scanner.Err()
}
