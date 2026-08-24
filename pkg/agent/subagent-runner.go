package agent

import (
	"context"
	"fmt"
	"mncode/pkg/provider"
	"mncode/pkg/tools"
	"strings"
	"time"
)

// SubagentRunner executes a specialized task using a subagent definition and autonomous ReAct loop
type SubagentRunner struct {
	ParentSession *Session
}

// Run executes a subagent with a dedicated sub-session and multi-turn tool execution
func (sr *SubagentRunner) Run(ctx context.Context, agentName, prompt string) (string, error) {
	subDef, ok := sr.ParentSession.Catalog.Agents[agentName]
	systemPrompt := ""
	roleName := agentName
	if ok {
		systemPrompt = subDef.Prompt
		roleName = subDef.Role
	} else {
		systemPrompt = fmt.Sprintf("You are a specialized subagent '%s'. Your task is to investigate, analyze, or execute the assigned subtask thoroughly and return a concise, actionable report.", agentName)
	}

	worktreeBase := sr.ParentSession.Config.GetSetting("worktree_base", "current")
	systemPrompt += fmt.Sprintf("\n[Subagent Isolation & Worktree Context: Base ref is '%s']\n", worktreeBase)

	subID := fmt.Sprintf("%s-%d", agentName, time.Now().Unix()%10000)
	if sr.ParentSession.Subagents != nil {
		sr.ParentSession.Subagents.Register(subID, agentName, roleName, prompt)
	}

	if sr.ParentSession.UI != nil {
		sr.ParentSession.UI.OnSubagentStart(agentName, roleName, prompt)
	}

	// Subagent tool registry
	subTools := tools.DefaultRegistry(sr.ParentSession.WorkspaceDir, true)
	subSession := &Session{
		ID:           subID,
		WorkspaceDir: sr.ParentSession.WorkspaceDir,
		Config:       sr.ParentSession.Config,
		Provider:     sr.ParentSession.Provider,
		Tools:        subTools,
		Catalog:      sr.ParentSession.Catalog,
		Accounts:     sr.ParentSession.Accounts,
		Router:       sr.ParentSession.Router,
		Tracker:      sr.ParentSession.Tracker,
		Subagents:    sr.ParentSession.Subagents,
		History:      nil,
		UI:           nil, // Silent subagent
	}

	// Multi-turn ReAct Loop for Subagent (up to 10 iterations)
	subSession.History = append(subSession.History, provider.Message{
		Role:    provider.RoleUser,
		Content: prompt,
	})

	maxTurns := 10
	var finalContent strings.Builder
	var lastErr error

	for turn := 0; turn < maxTurns; turn++ {
		req := &provider.CompletionRequest{
			SystemPrompt:   systemPrompt,
			Messages:       subSession.History,
			Tools:          subSession.getToolDefinitions(),
			Model:          subSession.Config.Model,
			MaxTokens:      subSession.Config.MaxTokens,
			ThinkingBudget: subSession.Config.ThinkingBudget,
			Temperature:    0.7,
		}

		resp, err := subSession.Provider.Stream(ctx, req, func(ev provider.StreamEvent) error {
			return nil
		})
		if err != nil {
			lastErr = fmt.Errorf("subagent %s error on turn %d: %w", agentName, turn+1, err)
			break
		}
		notifyUsage(sr.ParentSession.UI, resp.InputTokens, resp.OutputTokens, resp.ThinkingTokens)

		if subSession.Tracker != nil {
			inTokens := resp.InputTokens
			outTokens := resp.OutputTokens
			if inTokens == 0 && outTokens == 0 {
				inTokens = (len(systemPrompt) + len(prompt)) / 4
				outTokens = len(resp.Content) / 4
			}
			subSession.Tracker.Record(subSession.Config.Model, string(subSession.Config.Provider), inTokens, outTokens)
		}

		if resp.Content != "" {
			if finalContent.Len() > 0 {
				finalContent.WriteString("\n")
			}
			finalContent.WriteString(resp.Content)
			if sr.ParentSession.Subagents != nil {
				sr.ParentSession.Subagents.AddLog(subID, resp.Content)
			}
		}

		assistantMsg := provider.Message{
			Role:      provider.RoleAssistant,
			Content:   resp.Content,
			Thinking:  resp.Thinking,
			ToolCalls: resp.ToolCalls,
		}
		subSession.History = append(subSession.History, assistantMsg)

		if len(resp.ToolCalls) == 0 {
			break
		}

		for _, tc := range resp.ToolCalls {
			if sr.ParentSession.Subagents != nil {
				sr.ParentSession.Subagents.AddToolCall(subID, tc.Name)
			}
			toolResult := subSession.executeToolCall(ctx, &tc)
			subSession.History = append(subSession.History, provider.Message{
				Role:        provider.RoleTool,
				ToolResults: []provider.ToolResult{toolResult},
			})
		}
	}

	resultText := strings.TrimSpace(finalContent.String())
	if resultText == "" && lastErr == nil {
		resultText = fmt.Sprintf("Subagent %s completed assigned task.", agentName)
	}

	statsSuffix := ""
	if sr.ParentSession.Subagents != nil {
		sr.ParentSession.Subagents.Complete(subID, resultText, lastErr != nil)
		for _, rec := range sr.ParentSession.Subagents.List() {
			if rec.ID == subID {
				secs := int(rec.Duration.Seconds())
				dur := fmt.Sprintf("%ds", secs)
				if secs == 0 {
					dur = fmt.Sprintf("%dms", rec.Duration.Milliseconds())
				}
				statsSuffix = fmt.Sprintf("(%s · %d tool calls)", dur, len(rec.ToolCalls))
				break
			}
		}
	}

	if sr.ParentSession.UI != nil {
		sr.ParentSession.UI.OnSubagentComplete(agentName, statsSuffix)
	}

	return resultText, lastErr
}
