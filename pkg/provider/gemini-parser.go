package provider

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func (g *GeminiProvider) parseSSE(r io.Reader, cb func(StreamEvent) error) (*CompletionResponse, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	resp := &CompletionResponse{}
	toolIndex := 0

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if usage, ok := chunk["usageMetadata"].(map[string]interface{}); ok {
			if value, ok := usage["promptTokenCount"].(float64); ok {
				resp.InputTokens = int(value)
			}
			if value, ok := usage["candidatesTokenCount"].(float64); ok {
				resp.OutputTokens = int(value)
			}
			if value, ok := usage["thoughtsTokenCount"].(float64); ok {
				resp.ThinkingTokens = int(value)
			}
		}
		candidates, _ := chunk["candidates"].([]interface{})
		if len(candidates) == 0 {
			continue
		}
		candidate, _ := candidates[0].(map[string]interface{})
		content, _ := candidate["content"].(map[string]interface{})
		if content == nil {
			continue
		}
		parts, _ := content["parts"].([]interface{})
		for _, item := range parts {
			part, _ := item.(map[string]interface{})
			if text, ok := part["text"].(string); ok && text != "" {
				resp.Content += text
				if err := emitEvent(cb, StreamEvent{Type: EventToken, Text: text}); err != nil {
					return nil, err
				}
			}
			if functionCall, ok := part["functionCall"].(map[string]interface{}); ok {
				name, _ := functionCall["name"].(string)
				args, _ := functionCall["args"].(map[string]interface{})
				toolIndex++
				tc := ToolCall{ID: fmt.Sprintf("call_%d", toolIndex), Name: name, Arguments: args}
				resp.ToolCalls = append(resp.ToolCalls, tc)
				if err := emitEvent(cb, StreamEvent{Type: EventToolCallStart, ToolCall: &tc}); err != nil {
					return nil, err
				}
				if err := emitEvent(cb, StreamEvent{Type: EventToolCallComplete, ToolCall: &tc}); err != nil {
					return nil, err
				}
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
