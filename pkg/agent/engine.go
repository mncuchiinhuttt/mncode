package agent

import (
	"context"
	"fmt"
	"strings"

	"mncode/pkg/config"
	"mncode/pkg/provider"
)

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

	s.History = append(s.History, provider.Message{
		Role:    provider.RoleUser,
		Content: cleanedInput,
		Images:  images,
	})

	// Auto-compact safeguard if enabled and context exceeds 85%
	if s.Config.GetSetting("auto_compact", "true") == "true" {
		if usage := s.GetContextUsage(); usage.PercentUsed >= 85.0 && len(s.History) > 4 {
			_, _ = s.CompactHistory(ctx)
		}
	}

	for {
		toolDefs := s.getToolDefinitions()
		req := &provider.CompletionRequest{
			SystemPrompt:   s.BuildSystemPrompt(),
			Messages:       s.History,
			Tools:          toolDefs,
			Model:          s.Config.Model,
			MaxTokens:      s.Config.MaxTokens,
			ThinkingBudget: s.Config.ThinkingBudget,
			Temperature:    s.Config.Temperature,
		}

		resp, err := s.Provider.Stream(ctx, req, func(ev provider.StreamEvent) error {
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
					s.Remote.PushTerminalOutput(fmt.Sprintf("\n⚡ [Tool Executing] %s\n", ev.ToolCall.Name))
				}
			case provider.EventError:
				s.UI.OnError(ev.Error)
				if s.Remote != nil && s.Remote.IsActive && ev.Error != nil {
					s.Remote.PushTerminalOutput(fmt.Sprintf("\n🛑 [Error] %v\n", ev.Error))
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
		s.History = append(s.History, assistantMsg)

		// If no tools requested, turn is complete
		if len(resp.ToolCalls) == 0 {
			break
		}

		// Execute tool calls
		var results []provider.ToolResult
		for _, tc := range resp.ToolCalls {
			toolResult := s.executeToolCall(ctx, &tc)
			results = append(results, toolResult)
		}

		// Record Tool Results
		toolMsg := provider.Message{
			Role:        provider.RoleTool,
			ToolResults: results,
		}
		s.History = append(s.History, toolMsg)

		// Inject real-time steer directives into history for next reasoning step
		if steers := s.DrainSteer(); len(steers) > 0 {
			steerText := strings.Join(steers, "\n")
			s.History = append(s.History, provider.Message{
				Role:    provider.RoleUser,
				Content: fmt.Sprintf("[User Steering Directive (High Priority)]:\n%s", steerText),
			})
		}
	}

	_ = s.Save()
	return nil
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
			cleanTarget := strings.ReplaceAll(target, "\\", "/")
			if !strings.Contains(cleanTarget, "plans/") && !strings.Contains(cleanTarget, "reports/") {
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

	if s.UI != nil && !s.Config.AutoApprove {
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
	var list []provider.ToolDefinition
	for _, t := range s.Tools.All() {
		list = append(list, provider.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.Schema(),
		})
	}
	return list
}
