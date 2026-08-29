package agent

import (
	"context"
	"fmt"
	"mncode/pkg/provider"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ProcessDeepResearch executes an autonomous multi-turn deep research or literature review pipeline
func (s *Session) ProcessDeepResearch(ctx context.Context, topic string, isLitReview bool) (string, error) {
	if err := s.EnsureProvider(); err != nil {
		return "", err
	}

	slug := slugifyTopic(topic)
	timestamp := time.Now().Format("20060102-150405")
	reportsDir := filepath.Join(s.WorkspaceDir, "reports")
	_ = os.MkdirAll(reportsDir, 0755)

	targetFilename := fmt.Sprintf("research-%s-%s.md", timestamp, slug)
	pipelineName := "Deep Research Pipeline"
	if isLitReview {
		targetFilename = fmt.Sprintf("lit-review-%s-%s.md", timestamp, slug)
		pipelineName = "Academic Literature Review Pipeline"
	}
	targetFilePath := filepath.Join(reportsDir, targetFilename)

	if s.UI != nil {
		s.UI.OnQueryStart()
	}

	researchPrompt := fmt.Sprintf("[%s ACTIVATED]\nTopic: %s\nTarget Report File: %s\n\nExecute autonomous deep research:\n1. Decompose the topic into 3-5 key search inquiries.\n2. Use search_web to find authoritative sources, documentation, benchmarks, or academic papers.\n3. Use read_url_content to deeply read key pages and extract quotes/data.\n4. Write an exhaustive, publication-grade markdown report to '%s' using write_to_file.\n5. Ensure comprehensive comparison tables, architectural diagrams, trade-offs, and citations with full URLs.\n6. Provide a concise executive briefing when finished.", pipelineName, topic, targetFilePath, targetFilePath)
	ensureSessionIdentity(s)

	appendHistory(s, provider.Message{
		Role:    provider.RoleUser,
		Content: researchPrompt,
	})

	maxTurns := 18
	completed := false
	for turn := 0; turn < maxTurns; turn++ {
		toolDefs := s.getToolDefinitions()

		systemPrompt := s.BuildSystemPrompt() + fmt.Sprintf("\n\n[AUTONOMOUS RESEARCHER MODE]\nYou are a Principal Research Scientist & Systems Architect.\nConduct exhaustive, deep research on: %s.\nUse search_web and read_url_content extensively to gather real facts.\nSave full synthesized report to '%s'.", topic, targetFilePath)

		req := &provider.CompletionRequest{
			SystemPrompt:   systemPrompt,
			Messages:       historySnapshot(s),
			Tools:          toolDefs,
			Model:          s.Config.Model,
			MaxTokens:      s.Config.MaxTokens,
			ThinkingBudget: s.Config.ThinkingBudget,
			Temperature:    0.3,
		}

		resp, err := s.streamProvider(ctx, req, func(ev provider.StreamEvent) error {
			if s.UI == nil {
				return nil
			}
			switch ev.Type {
			case provider.EventToken:
				s.UI.OnToken(ev.Text)
			case provider.EventThinking:
				s.UI.OnThinking(ev.Thinking)
			case provider.EventToolCallStart:
				s.UI.OnToolCallStart(ev.ToolCall)
			case provider.EventError:
				s.UI.OnError(ev.Error)
			}
			return nil
		})

		if err != nil {
			if s.UI != nil {
				s.UI.OnError(err)
			}
			return targetFilePath, err
		}
		notifyUsage(s.UI, resp.InputTokens, resp.OutputTokens, resp.ThinkingTokens)

		if s.Tracker != nil {
			inTokens := resp.InputTokens
			outTokens := resp.OutputTokens
			if inTokens == 0 && outTokens == 0 {
				inTokens = (len(systemPrompt) + len(topic)) / 4
				outTokens = len(resp.Content) / 4
			}
			s.Tracker.Record(s.Config.Model, string(s.Config.Provider), inTokens, outTokens)
		}

		assistantMsg := provider.Message{
			Role:      provider.RoleAssistant,
			Content:   resp.Content,
			Thinking:  resp.Thinking,
			ToolCalls: resp.ToolCalls,
		}
		appendHistory(s, assistantMsg)

		if len(resp.ToolCalls) == 0 {
			completed = true
			break
		}

		var results []provider.ToolResult
		for _, tc := range resp.ToolCalls {
			toolResult := s.executeToolCall(ctx, &tc)
			results = append(results, toolResult)
		}

		toolMsg := provider.Message{
			Role:        provider.RoleTool,
			ToolResults: results,
		}
		appendHistory(s, toolMsg)
	}

	if !completed {
		if err := ctx.Err(); err != nil {
			return targetFilePath, err
		}
		err := fmt.Errorf("research reached the maximum of %d iterations before completion", maxTurns)
		if s.UI != nil {
			s.UI.OnError(err)
		}
		return targetFilePath, err
	}

	return targetFilePath, nil
}

func slugifyTopic(topic string) string {
	lower := strings.ToLower(topic)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := re.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 40 {
		slug = slug[:40]
	}
	if slug == "" {
		slug = "topic"
	}
	return slug
}
