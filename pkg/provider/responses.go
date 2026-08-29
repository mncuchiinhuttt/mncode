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
		return nil, newNetworkError(r.Name(), err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes := readProviderErrorBody(response.Body)
		return nil, newHTTPError(r.Name(), response.StatusCode, string(bodyBytes), response.Header)
	}
	return parseResponsesSSE(response.Body, cb)
}

func responsesInput(req *CompletionRequest) []map[string]interface{} {
	input := make([]map[string]interface{}, 0, len(req.Messages)*2+1)
	if req.SystemPrompt != "" {
		input = append(input, map[string]interface{}{
			"role": "developer",
			"content": []map[string]interface{}{{
				"type": "input_text",
				"text": req.SystemPrompt,
			}},
		})
	}
	for _, message := range req.Messages {
		// Responses function calls and their results are input items in their
		// own right; they must not be nested in a role/content message.
		if len(message.Content) > 0 || len(message.Images) > 0 {
			content := make([]map[string]interface{}, 0, 1+len(message.Images))
			if message.Content != "" {
				contentType := "input_text"
				if message.Role == RoleAssistant {
					contentType = "output_text"
				}
				content = append(content, map[string]interface{}{
					"type": contentType,
					"text": message.Content,
				})
			}
			for _, image := range message.Images {
				content = append(content, map[string]interface{}{
					"type": "input_image",
					"image_url": fmt.Sprintf("data:%s;base64,%s", image.MediaType, image.Data),
				})
			}
			input = append(input, map[string]interface{}{
				"role":    string(message.Role),
				"content": content,
			})
		}
		for _, toolCall := range message.ToolCalls {
			arguments := toolCall.RawArgs
			if arguments == "" {
				encoded, _ := json.Marshal(toolCall.Arguments)
				arguments = string(encoded)
			}
			input = append(input, map[string]interface{}{
				"type":      "function_call",
				"call_id":   toolCall.ID,
				"name":      toolCall.Name,
				"arguments": arguments,
			})
		}
		for _, toolResult := range message.ToolResults {
			input = append(input, map[string]interface{}{
				"type":    "function_call_output",
				"call_id": toolResult.ToolCallID,
				"output":  toolResult.Content,
			})
		}
	}
	return input
}

func parseResponsesSSE(reader io.Reader, cb func(StreamEvent) error) (*CompletionResponse, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	result := &CompletionResponse{}
	type responseTool struct {
		call     ToolCall
		started  bool
		complete bool
	}
	tools := make([]*responseTool, 0)
	byID := make(map[string]*responseTool)
	ensureTool := func(id string) *responseTool {
		if id != "" {
			if existing := byID[id]; existing != nil {
				return existing
			}
		}
		t := &responseTool{call: ToolCall{ID: id, Arguments: make(map[string]interface{})}}
		tools = append(tools, t)
		if id != "" {
			byID[id] = t
		}
		return t
	}
	bindToolID := func(id string, tool *responseTool) {
		if id != "" {
			byID[id] = tool
		}
	}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &data); err != nil {
			continue
		}
		typ, _ := data["type"].(string)
		if usage, ok := data["usage"].(map[string]interface{}); ok {
			if value, ok := usage["input_tokens"].(float64); ok {
				result.InputTokens = int(value)
			}
			if value, ok := usage["output_tokens"].(float64); ok {
				result.OutputTokens = int(value)
			}
			if value, ok := usage["output_tokens_details"].(map[string]interface{}); ok {
				if thought, ok := value["reasoning_tokens"].(float64); ok {
					result.ThinkingTokens = int(thought)
				}
			}
		}
		if response, ok := data["response"].(map[string]interface{}); ok {
			if usage, ok := response["usage"].(map[string]interface{}); ok {
				if value, ok := usage["input_tokens"].(float64); ok {
					result.InputTokens = int(value)
				}
				if value, ok := usage["output_tokens"].(float64); ok {
					result.OutputTokens = int(value)
				}
			}
		}
		if delta, ok := data["delta"].(string); ok {
			switch {
			case strings.Contains(typ, "output_text"):
				result.Content += delta
				if err := emitEvent(cb, StreamEvent{Type: EventToken, Text: delta}); err != nil {
					return nil, err
				}
			case strings.Contains(typ, "reasoning"):
				result.Thinking += delta
				if err := emitEvent(cb, StreamEvent{Type: EventThinking, Thinking: delta}); err != nil {
					return nil, err
				}
			case strings.Contains(typ, "function_call_arguments"):
				toolID, _ := data["item_id"].(string)
				t := ensureTool(toolID)
				t.call.RawArgs += delta
			}
		}
		switch typ {
		case "response.output_item.added", "response.output_item.created":
			item, _ := data["item"].(map[string]interface{})
			if item == nil {
				item = data
			}
			itemType, _ := item["type"].(string)
			if itemType == "function_call" {
				callID, _ := item["call_id"].(string)
				itemID, _ := item["id"].(string)
				id := callID
				if id == "" {
					id = itemID
				}
				t := ensureTool(id)
				bindToolID(callID, t)
				bindToolID(itemID, t)
				t.call.Name, _ = item["name"].(string)
				if args, ok := item["arguments"].(string); ok {
					t.call.RawArgs = args
				}
				if !t.started {
					t.started = true
					if err := emitEvent(cb, StreamEvent{Type: EventToolCallStart, ToolCall: &t.call}); err != nil {
						return nil, err
					}
				}
			}
		case "response.function_call_arguments.done":
			id, _ := data["item_id"].(string)
			t := ensureTool(id)
			if args, ok := data["arguments"].(string); ok {
				t.call.RawArgs = args
			}
			if !t.started {
				t.started = true
				if err := emitEvent(cb, StreamEvent{Type: EventToolCallStart, ToolCall: &t.call}); err != nil {
					return nil, err
				}
			}
			if !t.complete {
				t.complete = true
				if t.call.RawArgs != "" {
					_ = json.Unmarshal([]byte(t.call.RawArgs), &t.call.Arguments)
				}
				result.ToolCalls = append(result.ToolCalls, t.call)
				if err := emitEvent(cb, StreamEvent{Type: EventToolCallComplete, ToolCall: &t.call}); err != nil {
					return nil, err
				}
			}
		case "response.output_item.done":
			item, _ := data["item"].(map[string]interface{})
			if item == nil {
				item = data
			}
			itemType, _ := item["type"].(string)
			if itemType == "function_call" {
				callID, _ := item["call_id"].(string)
				itemID, _ := item["id"].(string)
				id := callID
				if id == "" {
					id = itemID
				}
				t := ensureTool(id)
				bindToolID(callID, t)
				bindToolID(itemID, t)
				if name, ok := item["name"].(string); ok {
					t.call.Name = name
				}
				if args, ok := item["arguments"].(string); ok {
					t.call.RawArgs = args
				}
				if !t.started {
					t.started = true
					if err := emitEvent(cb, StreamEvent{Type: EventToolCallStart, ToolCall: &t.call}); err != nil {
						return nil, err
					}
				}
				if !t.complete {
					t.complete = true
					if t.call.RawArgs != "" {
						_ = json.Unmarshal([]byte(t.call.RawArgs), &t.call.Arguments)
					}
					result.ToolCalls = append(result.ToolCalls, t.call)
					if err := emitEvent(cb, StreamEvent{Type: EventToolCallComplete, ToolCall: &t.call}); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for _, t := range tools {
		if t.complete {
			continue
		}
		if t.call.Name == "" && t.call.RawArgs == "" {
			continue
		}
		if !t.started {
			t.started = true
			if err := emitEvent(cb, StreamEvent{Type: EventToolCallStart, ToolCall: &t.call}); err != nil {
				return nil, err
			}
		}
		if t.call.RawArgs != "" {
			_ = json.Unmarshal([]byte(t.call.RawArgs), &t.call.Arguments)
		}
		result.ToolCalls = append(result.ToolCalls, t.call)
		if err := emitEvent(cb, StreamEvent{Type: EventToolCallComplete, ToolCall: &t.call}); err != nil {
			return nil, err
		}
	}
	if err := emitEvent(cb, StreamEvent{Type: EventDone}); err != nil {
		return nil, err
	}
	return result, nil
}
