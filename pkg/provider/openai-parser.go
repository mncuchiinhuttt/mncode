package provider

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
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
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	resp := &CompletionResponse{}
	toolCallsMap := make(map[int]*ToolCall)
	toolStarted := make(map[int]bool)

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
		if usage, ok := chunk["usage"].(map[string]interface{}); ok {
			if value, ok := usage["prompt_tokens"].(float64); ok {
				resp.InputTokens = int(value)
			}
			if value, ok := usage["completion_tokens"].(float64); ok {
				resp.OutputTokens = int(value)
			}
			if value, ok := usage["completion_tokens_details"].(map[string]interface{}); ok {
				if thought, ok := value["reasoning_tokens"].(float64); ok {
					resp.ThinkingTokens = int(thought)
				}
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
			if err := emitEvent(cb, StreamEvent{Type: EventToken, Text: content}); err != nil {
				return nil, err
			}
		}
		reasoning := ""
		if value, ok := delta["reasoning_content"].(string); ok {
			reasoning = value
		}
		if reasoning == "" {
			if value, ok := delta["reasoning"].(string); ok {
				reasoning = value
			}
		}
		if reasoning != "" {
			resp.Thinking += reasoning
			if err := emitEvent(cb, StreamEvent{Type: EventThinking, Thinking: reasoning}); err != nil {
				return nil, err
			}
		}
		if tcs, ok := delta["tool_calls"].([]interface{}); ok {
			for _, item := range tcs {
				tcMap, _ := item.(map[string]interface{})
				idx := 0
				if value, ok := tcMap["index"].(float64); ok {
					idx = int(value)
				}
				tc, exists := toolCallsMap[idx]
				if !exists {
					tc = &ToolCall{Arguments: make(map[string]interface{})}
					toolCallsMap[idx] = tc
				}
				if id, ok := tcMap["id"].(string); ok && id != "" {
					tc.ID = id
				}
				if fn, ok := tcMap["function"].(map[string]interface{}); ok {
					if name, ok := fn["name"].(string); ok && name != "" {
						tc.Name = name
						if !toolStarted[idx] {
							toolStarted[idx] = true
							if err := emitEvent(cb, StreamEvent{Type: EventToolCallStart, ToolCall: tc}); err != nil {
								return nil, err
							}
						}
					}
					if args, ok := fn["arguments"].(string); ok {
						tc.RawArgs += args
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	indices := make([]int, 0, len(toolCallsMap))
	for idx := range toolCallsMap {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	for _, idx := range indices {
		tc := toolCallsMap[idx]
		if tc.RawArgs != "" {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.RawArgs), &args); err == nil && args != nil {
				tc.Arguments = args
			}
		}
		resp.ToolCalls = append(resp.ToolCalls, *tc)
		if err := emitEvent(cb, StreamEvent{Type: EventToolCallComplete, ToolCall: tc}); err != nil {
			return nil, err
		}
	}
	if err := emitEvent(cb, StreamEvent{Type: EventDone}); err != nil {
		return nil, err
	}
	return resp, nil
}
