package provider

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type antigravityUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	ThoughtsTokenCount   int `json:"thoughtsTokenCount"`
}

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
			part := map[string]interface{}{
				"functionCall": map[string]interface{}{
					"name": tc.Name,
					"args": tc.Arguments,
				},
			}
			if tc.ThoughtSignature != "" {
				part["thoughtSignature"] = tc.ThoughtSignature
			}
			parts = append(parts, part)
		}
		for _, tr := range m.ToolResults {
			parts = append(parts, map[string]interface{}{
				"functionResponse": map[string]interface{}{
					"name":     tr.Name,
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
	inputTokens, outputTokens, thinkingTokens := 0, 0, 0

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
				UsageMetadata antigravityUsageMetadata `json:"usageMetadata"`
				Candidates    []struct {
					Content struct {
						Parts []struct {
							Text                  string                 `json:"text"`
							FunctionCall          map[string]interface{} `json:"functionCall"`
							Thought               bool                   `json:"thought"`
							ThoughtSignature      string                 `json:"thoughtSignature"`
							ThoughtSignatureSnake string                 `json:"thought_signature"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			} `json:"response"`
			UsageMetadata antigravityUsageMetadata `json:"usageMetadata"`
			Candidates    []struct {
				Content struct {
					Parts []struct {
						Text                  string                 `json:"text"`
						FunctionCall          map[string]interface{} `json:"functionCall"`
						Thought               bool                   `json:"thought"`
						ThoughtSignature      string                 `json:"thoughtSignature"`
						ThoughtSignatureSnake string                 `json:"thought_signature"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.UsageMetadata.PromptTokenCount > 0 {
			inputTokens = chunk.UsageMetadata.PromptTokenCount
		}
		if chunk.UsageMetadata.CandidatesTokenCount > 0 {
			outputTokens = chunk.UsageMetadata.CandidatesTokenCount
		}
		if chunk.UsageMetadata.ThoughtsTokenCount > 0 {
			thinkingTokens = chunk.UsageMetadata.ThoughtsTokenCount
		}
		if chunk.Response != nil {
			usage := chunk.Response.UsageMetadata
			if usage.PromptTokenCount > 0 {
				inputTokens = usage.PromptTokenCount
			}
			if usage.CandidatesTokenCount > 0 {
				outputTokens = usage.CandidatesTokenCount
			}
			if usage.ThoughtsTokenCount > 0 {
				thinkingTokens = usage.ThoughtsTokenCount
			}
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
					sig := part.ThoughtSignature
					if sig == "" {
						sig = part.ThoughtSignatureSnake
					}
					tc := ToolCall{
						ID:               fmt.Sprintf("tc-%d", time.Now().UnixNano()),
						Name:             name,
						Arguments:        args,
						ThoughtSignature: sig,
					}
					toolCalls = append(toolCalls, tc)
					_ = cb(StreamEvent{Type: EventToolCallStart, ToolCall: &tc})
				}
			}
		}
	}

	_ = cb(StreamEvent{Type: EventDone})
	return &CompletionResponse{
		Content:        fullText.String(),
		ToolCalls:      toolCalls,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
		ThinkingTokens: thinkingTokens,
	}, nil
}
