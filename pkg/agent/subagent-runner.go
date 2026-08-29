package agent

import (
	"context"
	"fmt"
	"mncode/pkg/orchestration"
	"mncode/pkg/provider"
	"mncode/pkg/tools"
	"strings"
	"time"
)

// SubagentRunner executes a specialized task using a subagent definition and autonomous ReAct loop
type SubagentRunner struct {
	ParentSession *Session
}

// Run executes a subagent with a dedicated sub-session and multi-turn tool execution.
func (sr *SubagentRunner) Run(ctx context.Context, agentName, prompt string) (output string, returnErr error) {
	if sr == nil || sr.ParentSession == nil {
		return "", fmt.Errorf("parent session is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	parent := sr.ParentSession
	subConfig, err := cloneSessionConfig(parent.Config)
	if err != nil {
		return "", err
	}
	subProvider, err := isolatedSubagentProvider(parent, subConfig)
	if err != nil {
		return "", err
	}
	systemPrompt := ""
	roleName := agentName
	if parent.Catalog != nil {
		if subDef, ok := parent.Catalog.Agents[agentName]; ok {
			systemPrompt = subDef.Prompt
			roleName = subDef.Role
		}
	}
	if systemPrompt == "" {
		systemPrompt = fmt.Sprintf("You are a specialized subagent '%s'. Your task is to investigate, analyze, or execute the assigned subtask thoroughly and return a concise, actionable report.", agentName)
	}

	worktreeBase := subConfig.GetSetting("worktree_base", "current")
	subID := fmt.Sprintf("%s-%d", agentName, time.Now().UnixNano())
	if parent.Subagents != nil {
		parent.Subagents.Register(subID, agentName, roleName, prompt)
	}
	if parent.UI != nil {
		parent.UI.OnSubagentStart(agentName, roleName, prompt)
	}

	// Git worktree isolation: when worktree_base is "main" or "fresh" and the
	// workspace is a git repo, the subagent gets its own worktree + branch.
	effectiveWorkspaceDir := parent.WorkspaceDir
	worktree, worktreeErr := CreateSubagentWorktree(parent.WorkspaceDir, subID, worktreeBase)
	if worktreeErr != nil {
		systemPrompt += fmt.Sprintf("\n[Subagent Isolation: requested worktree base '%s' but isolation is unavailable (%v) — operating directly on the shared workspace instead.]\n", worktreeBase, worktreeErr)
	} else if worktree != nil {
		effectiveWorkspaceDir = worktree.Path
		defer func() {
			if cleanupErr := worktree.Cleanup(parent.WorkspaceDir); cleanupErr != nil {
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

	// Subagent-owned process tools must not outlive this run.
	subTools := tools.DefaultRegistry(effectiveWorkspaceDir, true, subConfig)
	defer subTools.Close()
	subSession := &Session{
		ID: subID, WorkspaceDir: effectiveWorkspaceDir, Config: subConfig, Provider: subProvider,
		Tools: subTools, Catalog: parent.Catalog, Tracker: parent.Tracker,
		Subagents: parent.Subagents, History: nil, UI: newSubagentUI(parent.UI),
	}
	if parent.Subagents != nil {
		subTools.RegisterSpec(tools.ToolSpec{
			Tool:    &tools.SubagentPeerTool{SelfID: subID, Hub: parent.Subagents},
			Toolset: "subagents", Scope: tools.ScopeSession,
		})
	}

	// Multi-turn ReAct Loop for Subagent (up to 10 iterations)
	ensureSessionIdentity(subSession)
	appendHistory(subSession, provider.Message{
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
			Messages:       historySnapshot(subSession),
			Tools:          subSession.getToolDefinitions(),
			Model:          subSession.Config.Model,
			MaxTokens:      subSession.Config.MaxTokens,
			ThinkingBudget: subSession.Config.ThinkingBudget,
			Temperature:    0.7,
		}

		resp, err := subSession.streamProvider(ctx, req, func(ev provider.StreamEvent) error {
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
		appendHistory(subSession, assistantMsg)

		if len(resp.ToolCalls) == 0 {
			completed = true
			break
		}

		for _, tc := range resp.ToolCalls {
			if sr.ParentSession.Subagents != nil {
				sr.ParentSession.Subagents.AddToolCall(subID, tc.Name)
			}
			toolResult := subSession.executeToolCall(ctx, &tc)
			appendHistory(subSession, provider.Message{
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

// RunAsync spawns the subagent in background supervised by RunManager and returns the Run handle.
func (sr *SubagentRunner) RunAsync(ctx context.Context, agentName, prompt string, mgr *orchestration.RunManager) (*orchestration.Run, error) {
	if mgr == nil {
		return nil, fmt.Errorf("run manager is required for async subagent execution")
	}
	parentID := ""
	workspaceDir := ""
	if sr.ParentSession != nil {
		parentID = sr.ParentSession.ID
		workspaceDir = sr.ParentSession.WorkspaceDir
	}

	run, err := mgr.CreateRun(ctx, orchestration.RunMeta{
		ID:           fmt.Sprintf("subagent-%s-%d", agentName, time.Now().UnixNano()),
		ChatID:       parentID,
		ParentRunID:  parentID,
		Kind:         orchestration.KindSubagent,
		WorkspaceDir: workspaceDir,
		Labels: map[string]string{
			"agent": agentName,
		},
		Metadata: map[string]interface{}{
			"prompt": prompt,
		},
	})
	if err != nil {
		return nil, err
	}

	go func() {
		_ = run.Transition(orchestration.StateRunning)
		run.EmitEvent("subagent_started", map[string]string{
			"agent":  agentName,
			"prompt": prompt,
		})
		run.Log("Subagent %s started", agentName)

		output, err := sr.Run(run.Context(), agentName, prompt)
		if err != nil {
			run.Log("Subagent %s failed: %v", agentName, err)
			_ = run.Fail(err)
			run.EmitEvent("subagent_failed", map[string]string{
				"agent": agentName,
				"error": err.Error(),
			})
		} else {
			run.Log("Subagent %s completed", agentName)
			_ = run.Complete(output, 0, 0)
			run.EmitEvent("subagent_completed", map[string]string{
				"agent":  agentName,
				"output": output,
			})
		}
	}()

	return run, nil
}
