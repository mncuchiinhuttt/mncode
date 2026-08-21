package provider

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

func (a *AntigravityProvider) buildGeminiRequest(req *CompletionRequest) map[string]interface{} {
	payload := map[string]interface{}{}

	if req.SystemPrompt != "" {
		payload["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{{"text": req.SystemPrompt}},
		}
	}

	var contents []map[string]interface{}
	for _, m := range req.Messages {
		role := "user"
		if m.Role == RoleAssistant {
			role = "model"
		}
		var parts []map[string]interface{}
		if m.Content != "" {
			parts = append(parts, map[string]interface{}{"text": m.Content})
		}
		for _, tc := range m.ToolCalls {
			parts = append(parts, map[string]interface{}{
				"functionCall": map[string]interface{}{
					"name": tc.Name,
					"args": tc.Arguments,
				},
			})
		}
		for _, tr := range m.ToolResults {
			parts = append(parts, map[string]interface{}{
				"functionResponse": map[string]interface{}{
					"name": tr.Name,
					"response": map[string]interface{}{"content": tr.Content},
				},
			})
		}
		contents = append(contents, map[string]interface{}{"role": role, "parts": parts})
	}
	payload["contents"] = contents

	if len(req.Tools) > 0 {
		var decls []map[string]interface{}
		for _, t := range req.Tools {
			decls = append(decls, map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			})
		}
		payload["tools"] = []map[string]interface{}{{"functionDeclarations": decls}}
	}

	return payload
}

func (a *AntigravityProvider) parseSSE(r io.Reader, cb func(StreamEvent) error) (*CompletionResponse, error) {
	scanner := bufio.NewScanner(r)
	fullText := strings.Builder{}
	var toolCalls []ToolCall

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Response *struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text         string                 `json:"text"`
							FunctionCall map[string]interface{} `json:"functionCall"`
							Thought      bool                   `json:"thought"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			} `json:"response"`
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text         string                 `json:"text"`
						FunctionCall map[string]interface{} `json:"functionCall"`
						Thought      bool                   `json:"thought"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		candidates := chunk.Candidates
		if chunk.Response != nil && len(chunk.Response.Candidates) > 0 {
			candidates = chunk.Response.Candidates
		}

		for _, cand := range candidates {
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					if part.Thought {
						_ = cb(StreamEvent{Type: EventThinking, Thinking: part.Text})
					} else {
						fullText.WriteString(part.Text)
						_ = cb(StreamEvent{Type: EventToken, Text: part.Text})
					}
				}
				if len(part.FunctionCall) > 0 {
					name, _ := part.FunctionCall["name"].(string)
					args, _ := part.FunctionCall["args"].(map[string]interface{})
					tc := ToolCall{
						ID:        fmt.Sprintf("tc-%d", time.Now().UnixNano()),
						Name:      name,
						Arguments: args,
					}
					toolCalls = append(toolCalls, tc)
					_ = cb(StreamEvent{Type: EventToolCallStart, ToolCall: &tc})
				}
			}
		}
	}

	_ = cb(StreamEvent{Type: EventDone})
	return &CompletionResponse{
		Content:   fullText.String(),
		ToolCalls: toolCalls,
	}, nil
}
