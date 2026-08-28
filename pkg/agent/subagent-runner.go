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
func (sr *SubagentRunner) Run(ctx context.Context, agentName, prompt string) (output string, returnErr error) {
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

	subID := fmt.Sprintf("%s-%d", agentName, time.Now().UnixNano())
	if sr.ParentSession.Subagents != nil {
		sr.ParentSession.Subagents.Register(subID, agentName, roleName, prompt)
	}

	if sr.ParentSession.UI != nil {
		sr.ParentSession.UI.OnSubagentStart(agentName, roleName, prompt)
	}

	// Git worktree isolation: when worktree_base is "main" or "fresh" and the
	// workspace is a git repo, the subagent gets its own worktree + branch so
	// it can never race the user's own uncommitted edits or a sibling
	// subagent's concurrent file changes. Falls back to sharing the parent's
	// workspace directly (prior behavior) if isolation isn't applicable.
	effectiveWorkspaceDir := sr.ParentSession.WorkspaceDir
	worktree, worktreeErr := CreateSubagentWorktree(sr.ParentSession.WorkspaceDir, subID, worktreeBase)
	if worktreeErr != nil {
		systemPrompt += fmt.Sprintf("\n[Subagent Isolation: requested worktree base '%s' but isolation is unavailable (%v) — operating directly on the shared workspace instead.]\n", worktreeBase, worktreeErr)
	} else if worktree != nil {
		effectiveWorkspaceDir = worktree.Path
		defer func() {
			if cleanupErr := worktree.Cleanup(sr.ParentSession.WorkspaceDir); cleanupErr != nil {
				if returnErr == nil {
					returnErr = cleanupErr
				}
				output = strings.TrimSpace(output + fmt.Sprintf("\n\n[Worktree cleanup failed; recover changes from %s: %v]", worktree.Path, cleanupErr))
			}
		}()
		systemPrompt += fmt.Sprintf("\n[Subagent Isolation & Worktree Context: operating in an isolated git worktree on branch '%s', checked out from '%s'. Changes here do not affect the user's working directory until merged.]\n", worktree.Branch, worktree.BaseRef)
	} else {
		systemPrompt += fmt.Sprintf("\n[Subagent Isolation & Worktree Context: Base ref is '%s' (no isolation — operating on the shared workspace).]\n", worktreeBase)
	}

	// Subagent tool registry
	subTools := tools.DefaultRegistry(effectiveWorkspaceDir, true, sr.ParentSession.Config)
	subSession := &Session{
		ID:           subID,
		WorkspaceDir: effectiveWorkspaceDir,
		Config:       sr.ParentSession.Config,
		Provider:     sr.ParentSession.Provider,
		Tools:        subTools,
		Catalog:      sr.ParentSession.Catalog,
		Accounts:     sr.ParentSession.Accounts,
		Router:       sr.ParentSession.Router,
		Tracker:      sr.ParentSession.Tracker,
		Subagents:    sr.ParentSession.Subagents,
		History:      nil,
		UI:           newSubagentUI(sr.ParentSession.UI),
	}

	// Multi-turn ReAct Loop for Subagent (up to 10 iterations)
	subSession.History = append(subSession.History, provider.Message{
		Role:    provider.RoleUser,
		Content: prompt,
	})

	maxTurns := 10
	var finalContent strings.Builder
	var lastErr error
	completed := false

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
			completed = true
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

	if !completed && lastErr == nil {
		if err := ctx.Err(); err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("subagent %s reached the maximum of %d iterations before completion", agentName, maxTurns)
		}
	}

	resultText := strings.TrimSpace(finalContent.String())
	if resultText == "" && lastErr == nil {
		resultText = fmt.Sprintf("Subagent %s completed assigned task.", agentName)
	}
	if worktree != nil {
		resultText = strings.TrimSpace(resultText + fmt.Sprintf("\n\n[Isolated worktree preserved on branch %s. Review or merge this branch from the parent repository.]", worktree.Branch))
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
