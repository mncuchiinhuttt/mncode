package provider

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

func formatAnthropicMessage(m Message) map[string]interface{} {
	role := "user"
	if m.Role == RoleAssistant {
		role = "assistant"
	}

	var content []map[string]interface{}
	for _, img := range m.Images {
		content = append(content, map[string]interface{}{
			"type": "image",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": img.MediaType,
				"data":       img.Data,
			},
		})
	}
	if m.Content != "" {
		content = append(content, map[string]interface{}{
			"type": "text",
			"text": m.Content,
		})
	}
	for _, tc := range m.ToolCalls {
		content = append(content, map[string]interface{}{
			"type":  "tool_use",
			"id":    tc.ID,
			"name":  tc.Name,
			"input": tc.Arguments,
		})
	}
	for _, tr := range m.ToolResults {
		content = append(content, map[string]interface{}{
			"type":        "tool_result",
			"tool_use_id": tr.ToolCallID,
			"content":     tr.Content,
			"is_error":    tr.IsError,
		})
	}

	return map[string]interface{}{
		"role":    role,
		"content": content,
	}
}

func (a *AnthropicProvider) parseSSE(r io.Reader, cb func(StreamEvent) error) (*CompletionResponse, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	resp := &CompletionResponse{}
	var currentToolCall *ToolCall
	var rawArgs strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		eventType, _ := event["type"].(string)
		switch eventType {
		case "message_start":
			if msg, ok := event["message"].(map[string]interface{}); ok {
				if usage, ok := msg["usage"].(map[string]interface{}); ok {
					if value, ok := usage["input_tokens"].(float64); ok {
						resp.InputTokens = int(value)
					}
				}
			}
		case "message_delta":
			if usage, ok := event["usage"].(map[string]interface{}); ok {
				if value, ok := usage["output_tokens"].(float64); ok {
					resp.OutputTokens = int(value)
				}
			}
		case "content_block_start":
			if block, ok := event["content_block"].(map[string]interface{}); ok {
				if typ, _ := block["type"].(string); typ == "tool_use" {
					id, _ := block["id"].(string)
					name, _ := block["name"].(string)
					currentToolCall = &ToolCall{ID: id, Name: name, Arguments: make(map[string]interface{})}
					rawArgs.Reset()
					if err := emitEvent(cb, StreamEvent{Type: EventToolCallStart, ToolCall: currentToolCall}); err != nil {
						return nil, err
					}
				}
			}
		case "content_block_delta":
			if delta, ok := event["delta"].(map[string]interface{}); ok {
				switch typ, _ := delta["type"].(string); typ {
				case "text_delta":
					txt, _ := delta["text"].(string)
					if txt != "" {
						resp.Content += txt
						if err := emitEvent(cb, StreamEvent{Type: EventToken, Text: txt}); err != nil {
							return nil, err
						}
					}
				case "thinking_delta":
					th, _ := delta["thinking"].(string)
					if th != "" {
						resp.Thinking += th
						if err := emitEvent(cb, StreamEvent{Type: EventThinking, Thinking: th}); err != nil {
							return nil, err
						}
					}
				case "input_json_delta":
					partial, _ := delta["partial_json"].(string)
					rawArgs.WriteString(partial)
				}
			}
		case "content_block_stop":
			if currentToolCall != nil {
				currentToolCall.RawArgs = rawArgs.String()
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(currentToolCall.RawArgs), &args); err == nil && args != nil {
					currentToolCall.Arguments = args
				}
				resp.ToolCalls = append(resp.ToolCalls, *currentToolCall)
				if err := emitEvent(cb, StreamEvent{Type: EventToolCallComplete, ToolCall: currentToolCall}); err != nil {
					return nil, err
				}
				currentToolCall = nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := emitEvent(cb, StreamEvent{Type: EventDone}); err != nil {
		return nil, err
	}
	return resp, nil
}
