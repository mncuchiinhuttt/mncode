package provider

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func formatOpenAIMessage(m Message) map[string]interface{} {
	role := string(m.Role)
	if m.Role == RoleTool && len(m.ToolResults) > 0 {
		return map[string]interface{}{
			"role":         "tool",
			"tool_call_id": m.ToolResults[0].ToolCallID,
			"content":      m.ToolResults[0].Content,
		}
	}

	if len(m.Images) > 0 {
		var contentBlocks []map[string]interface{}
		if m.Content != "" {
			contentBlocks = append(contentBlocks, map[string]interface{}{
				"type": "text",
				"text": m.Content,
			})
		}
		for _, img := range m.Images {
			contentBlocks = append(contentBlocks, map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]interface{}{
					"url": fmt.Sprintf("data:%s;base64,%s", img.MediaType, img.Data),
				},
			})
		}
		return map[string]interface{}{"role": role, "content": contentBlocks}
	}

	res := map[string]interface{}{"role": role, "content": m.Content}
	if len(m.ToolCalls) > 0 {
		var tcs []map[string]interface{}
		for _, tc := range m.ToolCalls {
			argsBytes, _ := json.Marshal(tc.Arguments)
			tcs = append(tcs, map[string]interface{}{
				"id":   tc.ID,
				"type": "function",
				"function": map[string]interface{}{
					"name":      tc.Name,
					"arguments": string(argsBytes),
				},
			})
		}
		res["tool_calls"] = tcs
	}
	return res
}

func (o *OpenAIProvider) parseSSE(r io.Reader, cb func(StreamEvent) error) (*CompletionResponse, error) {
	scanner := bufio.NewScanner(r)
	resp := &CompletionResponse{}
	toolCallsMap := make(map[int]*ToolCall)

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

		// Extract OpenAI / OpenRouter token usage
		if usage, ok := chunk["usage"].(map[string]interface{}); ok {
			if pt, ok := usage["prompt_tokens"].(float64); ok {
				resp.InputTokens = int(pt)
			}
			if ct, ok := usage["completion_tokens"].(float64); ok {
				resp.OutputTokens = int(ct)
			}
		}

		choices, _ := chunk["choices"].([]interface{})
		if len(choices) == 0 {
			continue
		}

		choice, _ := choices[0].(map[string]interface{})
		delta, _ := choice["delta"].(map[string]interface{})
		if delta == nil {
			continue
		}

		if content, ok := delta["content"].(string); ok && content != "" {
			resp.Content += content
			_ = cb(StreamEvent{Type: EventToken, Text: content})
		}

		if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
			resp.Thinking += reasoning
			_ = cb(StreamEvent{Type: EventThinking, Thinking: reasoning})
		}

		if tcs, ok := delta["tool_calls"].([]interface{}); ok {
			for _, tcItem := range tcs {
				tcMap, _ := tcItem.(map[string]interface{})
				idx := 0
				if i, ok := tcMap["index"].(float64); ok {
					idx = int(i)
				}
				if _, exists := toolCallsMap[idx]; !exists {
					toolCallsMap[idx] = &ToolCall{Arguments: make(map[string]interface{})}
				}
				tc := toolCallsMap[idx]
				if id, ok := tcMap["id"].(string); ok && id != "" {
					tc.ID = id
				}
				if fn, ok := tcMap["function"].(map[string]interface{}); ok {
					if name, ok := fn["name"].(string); ok && name != "" {
						tc.Name = name
						_ = cb(StreamEvent{Type: EventToolCallStart, ToolCall: tc})
					}
					if argsChunk, ok := fn["arguments"].(string); ok {
						tc.RawArgs += argsChunk
					}
				}
			}
		}
	}

	for _, tc := range toolCallsMap {
		if tc.RawArgs != "" {
			var args map[string]interface{}
			_ = json.Unmarshal([]byte(tc.RawArgs), &args)
			tc.Arguments = args
		}
		resp.ToolCalls = append(resp.ToolCalls, *tc)
		_ = cb(StreamEvent{Type: EventToolCallComplete, ToolCall: tc})
	}

	_ = cb(StreamEvent{Type: EventDone})
	return resp, scanner.Err()
}
