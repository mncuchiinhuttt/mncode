package agent

import (
	"context"
	"fmt"
	"mncode/pkg/provider"
	"os"
	"path/filepath"
	"time"
)

// ProcessPlanGeneration executes the autonomous plan creation pipeline in ./plans/
func (s *Session) ProcessPlanGeneration(ctx context.Context, task string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.EnsureProvider(); err != nil {
		return "", err
	}

	slug := slugifyTopic(task)
	timestamp := time.Now().Format("20060102-1504")
	planFolderName := fmt.Sprintf("%s-%s", timestamp, slug)
	planDir := filepath.Join(s.WorkspaceDir, "plans", planFolderName)
	_ = os.MkdirAll(planDir, 0755)

	planOverviewPath := filepath.Join(planDir, "plan.md")
	phase1Path := filepath.Join(planDir, "phase-01-implementation.md")

	if s.UI != nil {
		s.UI.OnQueryStart()
	}

	planPrompt := fmt.Sprintf(`[AUTONOMOUS PLANNER MODE ACTIVATED]
Task / Feature: %s
Target Plan Directory: %s

Please research the codebase and create a production-grade multi-phase implementation plan:
1. Use view_file, grep_search, find_by_name to inspect existing architecture.
2. Create '%s' (Overview Plan, strictly under 80 lines with phase breakdown & status).
3. Create '%s' (Detailed Phase 1 file containing: Context Links, Overview, Key Insights, Requirements, Architecture, Related Code Files, Implementation Steps, Todo List checkboxes, Success Criteria, Risk Assessment, Security Considerations).
4. DO NOT modify any application code files; only write the plan documents to '%s'.
5. Provide a clear summary and next steps when completed.`, task, planDir, planOverviewPath, phase1Path, planDir)
	ensureSessionIdentity(s)

	appendHistory(s, provider.Message{
		Role:    provider.RoleUser,
		Content: planPrompt,
	})
	s.recordEvent("prompt", 0, planPrompt)

	maxTurns := 12
	completed := false
	for turn := 0; turn < maxTurns; turn++ {
		toolDefs := s.getToolDefinitions()

		systemPrompt := s.BuildSystemPrompt() + fmt.Sprintf("\n\n[PLANNER MODE]\nYou are a Principal Software Architect.\nCreate structured plan files in '%s'. You are strictly forbidden from modifying source files outside ./plans/.", planDir)

		req := &provider.CompletionRequest{
			SystemPrompt:   systemPrompt,
			Messages:       historySnapshot(s),
			Tools:          toolDefs,
			Model:          s.Config.Model,
			MaxTokens:      s.Config.MaxTokens,
			ThinkingBudget: s.Config.ThinkingBudget,
			Temperature:    0.2,
		}
		s.recordEvent("provider_request", turn+1, map[string]interface{}{"model": req.Model, "message_count": len(req.Messages), "tool_count": len(req.Tools)})

		resp, err := s.streamProvider(ctx, req, func(ev provider.StreamEvent) error {
			s.recordStreamEvent(turn+1, ev)
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
			s.recordEvent("error", turn+1, err.Error())
			if s.UI != nil {
				s.UI.OnError(err)
			}
			return planDir, err
		}
		s.recordEvent("provider_response", turn+1, map[string]interface{}{"content": resp.Content, "thinking": resp.Thinking, "tool_calls": resp.ToolCalls})
		notifyUsage(s.UI, resp.InputTokens, resp.OutputTokens, resp.ThinkingTokens)

		if s.Tracker != nil {
			inTokens := resp.InputTokens
			outTokens := resp.OutputTokens
			if inTokens == 0 && outTokens == 0 {
				inTokens = (len(systemPrompt) + len(task)) / 4
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
			s.recordEvent("turn_end", turn+1, map[string]interface{}{"completed": true})
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
		s.recordEvent("tool_result", turn+1, results)
	}

	if !completed {
		if err := ctx.Err(); err != nil {
			return planDir, err
		}
		err := fmt.Errorf("plan generation reached the maximum of %d iterations before completion", maxTurns)
		if s.UI != nil {
			s.UI.OnError(err)
		}
		return planDir, err
	}

	return planDir, nil
}
