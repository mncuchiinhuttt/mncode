package agent

import (
	"context"
	"fmt"
	"mncode/pkg/provider"
	"time"
)

// ProcessGoal runs an autonomous, persistent goal execution loop with live stopwatch
func (s *Session) ProcessGoal(ctx context.Context, goal string) error {
	if err := s.EnsureProvider(); err != nil {
		return err
	}

	startTime := time.Now()
	if s.UI != nil {
		s.UI.OnQueryStart()
	}

	goalPrompt := fmt.Sprintf("GOAL MODE: %s\n\nPlease execute autonomously. Use all necessary tools, verify changes, run tests, and only complete when the goal is verified 100%% achieved.", goal)
	ensureSessionIdentity(s)

	appendHistory(s, provider.Message{
		Role:    provider.RoleUser,
		Content: goalPrompt,
	})

	if usage := s.GetContextUsage(); usage.PercentUsed >= 85.0 && len(s.History) > 4 {
		_, _ = s.CompactHistory(ctx)
	}

	maxGoalTurns := 25
	totalToolCalls := 0
	turnCount := 0
	completed := false

	for turn := 0; turn < maxGoalTurns; turn++ {
		turnCount++
		toolDefs := s.getToolDefinitions()

		systemPrompt := s.BuildSystemPrompt() + fmt.Sprintf("\n\n[PERSISTENT GOAL MODE ACTIVE]\nCurrent Goal: %s\nYou must continue working autonomously until this goal is completely achieved and verified. Execute tools, run tests, create/edit code as needed.", goal)

		req := &provider.CompletionRequest{
			SystemPrompt:   systemPrompt,
			Messages:       historySnapshot(s),
			Tools:          toolDefs,
			Model:          s.Config.Model,
			MaxTokens:      s.Config.MaxTokens,
			ThinkingBudget: s.Config.ThinkingBudget,
			Temperature:    s.Config.Temperature,
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
			return err
		}
		notifyUsage(s.UI, resp.InputTokens, resp.OutputTokens, resp.ThinkingTokens)

		if s.Tracker != nil {
			inTokens := resp.InputTokens
			outTokens := resp.OutputTokens
			if inTokens == 0 && outTokens == 0 {
				inTokens = (len(systemPrompt) + len(goal)) / 4
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

		totalToolCalls += len(resp.ToolCalls)
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
			return err
		}
		err := fmt.Errorf("goal reached the maximum of %d iterations before completion", maxGoalTurns)
		if s.UI != nil {
			s.UI.OnError(err)
		}
		return err
	}

	elapsed := time.Since(startTime).Seconds()
	if s.UI != nil {
		s.UI.OnGoalDone(goal, elapsed, turnCount, totalToolCalls)
	}

	return nil
}
