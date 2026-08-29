package agent

import (
	"context"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mncode/pkg/accounts"
	"mncode/pkg/config"
	"mncode/pkg/provider"
	"mncode/pkg/tools"
)

// agentTurnLimit returns a bounded ReAct iteration count. The setting is
// intentionally capped so a malformed config cannot disable the safety guard.
func agentTurnLimit(cfg *config.Config) int {
	const defaultLimit = 25
	const maxLimit = 100
	if cfg == nil {
		return defaultLimit
	}
	value, err := strconv.Atoi(strings.TrimSpace(cfg.GetSetting("max_agent_turns", "25")))
	if err != nil || value <= 0 {
		return defaultLimit
	}
	if value > maxLimit {
		return maxLimit
	}
	return value
}

// ProcessUserInput executes an agent conversation turn with full tool-calling ReAct loop
func (s *Session) ProcessUserInput(ctx context.Context, userInput string) error {
	s.SetExecuting(true)
	defer s.SetExecuting(false)

	if err := s.EnsureProvider(); err != nil {
		return err
	}

	if s.UI != nil {
		s.UI.OnQueryStart()
	}

	userInput = s.PreprocessSkillTags(userInput)
	cleanedInput, images := ExtractImagesFromInput(s.WorkspaceDir, userInput)

	appendHistory(s, provider.Message{
		Role:    provider.RoleUser,
		Content: cleanedInput,
		Images:  images,
	})

	// Auto-compact safeguard if enabled and context exceeds 85%
	if s.Config.GetSetting("auto_compact", "true") == "true" {
		if usage := s.GetContextUsage(); usage.PercentUsed >= 85.0 && len(historySnapshot(s)) > 4 {
			if _, compactErr := s.CompactHistory(ctx); compactErr != nil && s.UI != nil {
				s.UI.OnError(compactErr)
			}
		}
	}
	if s.Budget != nil && s.Budget.IsHardStopExceeded() {
		return fmt.Errorf("session token budget hard limit reached. Use '/budget' to extend or clear")
	}

	maxTurns := agentTurnLimit(s.Config)
	completed := false
	for turn := 0; turn < maxTurns; turn++ {
		toolDefs := s.getToolDefinitions()
		req := &provider.CompletionRequest{
			SystemPrompt:   s.BuildSystemPrompt(),
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
				if s.Remote != nil && s.Remote.IsActive {
					s.Remote.PushTerminalOutput(ev.Text)
				}
			case provider.EventThinking:
				s.UI.OnThinking(ev.Thinking)
				if s.Remote != nil && s.Remote.IsActive {
					s.Remote.PushTerminalOutput(ev.Thinking)
				}
			case provider.EventToolCallStart:
				s.UI.OnToolCallStart(ev.ToolCall)
				if s.Remote != nil && s.Remote.IsActive && ev.ToolCall != nil {
					s.Remote.PushTerminalOutput(fmt.Sprintf("\n[ACTION] [Tool Executing] %s\n", ev.ToolCall.Name))
				}
			case provider.EventError:
				s.UI.OnError(ev.Error)
				if s.Remote != nil && s.Remote.IsActive && ev.Error != nil {
					s.Remote.PushTerminalOutput(fmt.Sprintf("\n[STOP] [Error] %v\n", ev.Error))
				}
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
		if s.Budget != nil {
			hardStop, notice := s.Budget.AddTokens(resp.InputTokens, resp.OutputTokens, resp.ThinkingTokens)
			if notice != "" {
				fmt.Printf("\n\033[1;33m%s\033[0m\n\n", notice)
			}
			if hardStop {
				return fmt.Errorf("session token budget exhausted. Aborting agent turn")
			}
		}

		if s.UI != nil {
			s.UI.Flush()
		}

		// Record token stats
		if s.Tracker != nil {
			inTokens := resp.InputTokens
			outTokens := resp.OutputTokens
			if inTokens == 0 && outTokens == 0 {
				inTokens = (len(req.SystemPrompt) + len(userInput)) / 4
				outTokens = len(resp.Content) / 4
			}
			s.Tracker.Record(s.Config.Model, string(s.Config.Provider), inTokens, outTokens)
		}

		// Record Assistant Message
		assistantMsg := provider.Message{
			Role:      provider.RoleAssistant,
			Content:   resp.Content,
			Thinking:  resp.Thinking,
			ToolCalls: resp.ToolCalls,
		}
		appendHistory(s, assistantMsg)

		// If no tools requested, turn is complete
		if len(resp.ToolCalls) == 0 {
			completed = true
			break
		}

		// Execute tool calls
		var results []provider.ToolResult
		for _, tc := range resp.ToolCalls {
			toolResult := s.executeToolCall(ctx, &tc)
			results = append(results, toolResult)
		}

		// Record Tool Results
		appendHistory(s, provider.Message{
			Role:        provider.RoleTool,
			ToolResults: results,
		})

		// Inject real-time steer directives into history for next reasoning step.
		if steers := s.DrainSteer(); len(steers) > 0 {
			steerText := strings.Join(steers, "\n")
			appendHistory(s, provider.Message{
				Role:    provider.RoleUser,
				Content: fmt.Sprintf("[User Steering Directive (High Priority)]:\n%s", steerText),
			})
		}
	}

	if !completed {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fmt.Errorf("agent turn reached the maximum of %d iterations; send a new prompt to continue", maxTurns)
		if s.UI != nil {
			s.UI.OnError(err)
		}
		return err
	}
	_ = s.Save()
	return nil
}

func planWriteTargetAllowed(workspace, rawPath string) bool {
	if strings.TrimSpace(workspace) == "" {
		return false
	}
	resolved, err := tools.ResolveWorkspacePath(workspace, rawPath, true)
	if err != nil {
		return false
	}
	root, err := filepath.Abs(workspace)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == "." || filepath.IsAbs(relative) {
		return false
	}
	relative = filepath.ToSlash(relative)
	return relative == "plans" || strings.HasPrefix(relative, "plans/") ||
		relative == "reports" || strings.HasPrefix(relative, "reports/")
}
func (s *Session) streamProvider(ctx context.Context, req *provider.CompletionRequest, cb func(provider.StreamEvent) error) (*provider.CompletionResponse, error) {
	const maxAttempts = 3
	for attempt := range maxAttempts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if s.Provider == nil {
			return nil, fmt.Errorf("provider is unavailable")
		}

		activeID := ""
		if identifiable, ok := s.Provider.(provider.AccountIdentifiable); ok {
			activeID = identifiable.AccountID()
		}
		scrubber := NewMemoryContextScrubber()
		attemptEvents := make([]provider.StreamEvent, 0, 16)
		streamCallback := func(event provider.StreamEvent) error {
			if event.Type == provider.EventToken {
				event.Text = scrubber.Feed(event.Text)
			} else if event.Type == provider.EventThinking {
				event.Thinking = scrubber.Feed(event.Thinking)
			}
			if event.Type == provider.EventToken && event.Text == "" {
				return nil
			}
			if event.Type == provider.EventThinking && event.Thinking == "" {
				return nil
			}
			if event.ToolCall != nil {
				toolCall := *event.ToolCall
				event.ToolCall = &toolCall
			}
			attemptEvents = append(attemptEvents, event)
			return nil
		}
		resp, err := s.Provider.Stream(ctx, req, streamCallback)
		if err == nil {
			if trailing := scrubber.Flush(); trailing != "" {
				attemptEvents = append(attemptEvents, provider.StreamEvent{Type: provider.EventToken, Text: trailing})
			}
			if cb != nil {
				for _, event := range attemptEvents {
					if callbackErr := cb(event); callbackErr != nil {
						return nil, callbackErr
					}
				}
			}
			if resp != nil {
				resp.Content = ScrubMemoryContext(resp.Content)
				resp.Thinking = ScrubMemoryContext(resp.Thinking)
			}
			if activeID != "" && s.Router != nil {
				s.Router.ReportSuccess(activeID)
			}
			return resp, nil
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		classified := provider.ClassifyError(err)
		if !classified.Retryable {
			return nil, err
		}
		if activeID != "" && s.Router != nil {
			s.Router.ReportFailure(activeID, classified.StatusCode, classified.Message)
		}
		if attempt == maxAttempts-1 {
			return nil, err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		// Refresh the current credential before rotating accounts for auth
		// failures. Other retryable errors rotate when account routing is
		// available, and otherwise retry the current provider.
		retryCurrent := false
		if refresher, ok := s.Provider.(provider.TokenRefresher); ok && (classified.StatusCode == 401 || classified.StatusCode == 403) {
			if _, refreshErr := refresher.RefreshTokenNow(); refreshErr == nil {
				retryCurrent = true
			}
		}
		if !retryCurrent && s.Router != nil {
			accountType := routedProviderType(s.Provider.Name())
			if accountType != "" {
				if next, nextErr := s.Router.GetNextAccount(accountType); nextErr == nil {
					if useErr := s.useAccount(next); useErr == nil {
						retryCurrent = true
					}
				}
			}
		}

		if delayErr := waitProviderRetry(ctx, attempt, classified.RetryAfter); delayErr != nil {
			return nil, delayErr
		}
	}
	return nil, ctx.Err()
}

func routedProviderType(name string) accounts.AccountProvider {
	switch strings.ToLower(name) {
	case "antigravity":
		return accounts.ProviderTypeAntigravity
	case "openai", "openrouter":
		return accounts.ProviderTypeCodex
	default:
		return ""
	}
}

func waitProviderRetry(ctx context.Context, attempt int, retryAfter time.Duration) error {
	delay := 100 * time.Millisecond
	for i := 0; i < attempt; i++ {
		delay *= 2
	}
	if retryAfter > delay {
		delay = retryAfter
	}
	if delay > 5*time.Second {
		delay = 5 * time.Second
	}
	jitter := time.Duration(rand.Int64N(int64(delay/2) + 1))
	timer := time.NewTimer(delay/2 + jitter)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *Session) executeToolCall(ctx context.Context, tc *provider.ToolCall) provider.ToolResult {
	// Strict Plan Mode enforcement: block code editing tools
	if s.Config.PermissionMode == config.PermissionModePlan || strings.EqualFold(s.Config.Workflow, "plan") {
		if tc.Name == "edit_file_content" || tc.Name == "replace_file_content" {
			errStr := "Plan Mode Block: Code editing is disabled in Plan Mode. You may only research and write plans in ./plans/."
			if s.UI != nil {
				s.UI.OnToolCallResult(tc.Name, errStr, true)
			}
			return provider.ToolResult{
				ToolCallID: tc.ID,
				Name:       tc.Name,
				Content:    errStr,
				IsError:    true,
			}
		}
		if tc.Name == "write_to_file" {
			target, _ := tc.Arguments["TargetFile"].(string)
			if target == "" {
				target, _ = tc.Arguments["path"].(string)
			}
			if !planWriteTargetAllowed(s.WorkspaceDir, target) {
				errStr := fmt.Sprintf("Plan Mode Block: Creating '%s' is disabled in Plan Mode. You may only create plan files in ./plans/.", target)
				if s.UI != nil {
					s.UI.OnToolCallResult(tc.Name, errStr, true)
				}
				return provider.ToolResult{
					ToolCallID: tc.ID,
					Name:       tc.Name,
					Content:    errStr,
					IsError:    true,
				}
			}
		}
	}

	if !s.Config.AutoApprove {
		if s.UI == nil {
			content := "Tool execution denied: approval is required but no approval UI is attached."
			return provider.ToolResult{
				ToolCallID: tc.ID,
				Name:       tc.Name,
				Content:    content,
				IsError:    true,
			}
		}
		if !s.UI.ConfirmToolExecution(tc) {
			res := provider.ToolResult{
				ToolCallID: tc.ID,
				Name:       tc.Name,
				Content:    "Tool execution cancelled by user.",
				IsError:    true,
			}
			s.UI.OnToolCallResult(tc.Name, res.Content, true)
			return res
		}
	}

	out, err := s.Tools.Execute(ctx, tc.Name, tc.Arguments)
	isErr := err != nil
	content := out
	if isErr && content == "" {
		content = fmt.Sprintf("Error: %v", err)
	}

	if s.UI != nil {
		s.UI.OnToolCallResult(tc.Name, content, isErr)
	}

	return provider.ToolResult{
		ToolCallID: tc.ID,
		Name:       tc.Name,
		Content:    content,
		IsError:    isErr,
	}
}

func (s *Session) getToolDefinitions() []provider.ToolDefinition {
	if s == nil || s.Tools == nil {
		return nil
	}
	definitions := s.Tools.Definitions(context.Background())
	list := make([]provider.ToolDefinition, 0, len(definitions))
	for _, definition := range definitions {
		list = append(list, provider.ToolDefinition{
			Name:        definition.Name,
			Description: definition.Description,
			InputSchema: definition.InputSchema,
		})
	}
	return list
}
