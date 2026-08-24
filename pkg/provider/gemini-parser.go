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
	resp := &CompletionResponse{}
	toolIndex := 0

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// Extract token usage metadata from Gemini
		if usage, ok := chunk["usageMetadata"].(map[string]interface{}); ok {
			if pt, ok := usage["promptTokenCount"].(float64); ok {
				resp.InputTokens = int(pt)
			}
			if ct, ok := usage["candidatesTokenCount"].(float64); ok {
				resp.OutputTokens = int(ct)
			}
			if tt, ok := usage["thoughtsTokenCount"].(float64); ok {
				resp.ThinkingTokens = int(tt)
			}
		}

		candidates, _ := chunk["candidates"].([]interface{})
		if len(candidates) == 0 {
			continue
		}

		cand, _ := candidates[0].(map[string]interface{})
		content, _ := cand["content"].(map[string]interface{})
		if content == nil {
			continue
		}

		parts, _ := content["parts"].([]interface{})
		for _, p := range parts {
			part, _ := p.(map[string]interface{})
			if txt, ok := part["text"].(string); ok && txt != "" {
				resp.Content += txt
				_ = cb(StreamEvent{Type: EventToken, Text: txt})
			}
			if fc, ok := part["functionCall"].(map[string]interface{}); ok {
				name, _ := fc["name"].(string)
				args, _ := fc["args"].(map[string]interface{})
				toolIndex++
				tc := ToolCall{
					ID:        fmt.Sprintf("call_%d", toolIndex),
					Name:      name,
					Arguments: args,
				}
				resp.ToolCalls = append(resp.ToolCalls, tc)
				_ = cb(StreamEvent{Type: EventToolCallComplete, ToolCall: &tc})
			}
		}
	}

	_ = cb(StreamEvent{Type: EventDone})
	return resp, scanner.Err()
}
