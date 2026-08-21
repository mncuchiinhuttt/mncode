package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"mncode/pkg/provider"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CompactionResult struct {
	OriginalTokens int
	CompactTokens  int
	FreedTokens    int
	PercentFreed   float64
	Summary        string
	SnapshotFile   string
}

// CompactHistory compresses long conversation history into a structured summary checkpoint
func (s *Session) CompactHistory(ctx context.Context) (*CompactionResult, error) {
	if len(s.History) <= 2 {
		return nil, fmt.Errorf("conversation history too short to compact (%d messages)", len(s.History))
	}

	beforeTokens := s.GetContextUsage().TotalUsed

	// 1. Save full snapshot before compacting
	home, _ := os.UserHomeDir()
	snapDir := filepath.Join(home, ".mncode", "snapshots")
	_ = os.MkdirAll(snapDir, 0755)
	snapFile := filepath.Join(snapDir, fmt.Sprintf("snapshot-%d.json", time.Now().Unix()))

	if snapData, err := json.MarshalIndent(s.History, "", "  "); err == nil {
		_ = os.WriteFile(snapFile, snapData, 0644)
	}

	// 2. Build summarization prompt
	var historyText strings.Builder
	for idx, msg := range s.History {
		role := strings.ToUpper(string(msg.Role))
		historyText.WriteString(fmt.Sprintf("\n--- Message %d (%s) ---\n", idx+1, role))
		if msg.Content != "" {
			historyText.WriteString(msg.Content + "\n")
		}
		for _, tc := range msg.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			historyText.WriteString(fmt.Sprintf("[Tool Call: %s(%s)]\n", tc.Name, string(argsJSON)))
		}
		for _, tr := range msg.ToolResults {
			contentPreview := tr.Content
			if len(contentPreview) > 200 {
				contentPreview = contentPreview[:200] + "..."
			}
			historyText.WriteString(fmt.Sprintf("[Tool Result %s: %s]\n", tr.Name, contentPreview))
		}
	}

	compactPrompt := fmt.Sprintf(`You are an expert AI Context Compactor.
Summarize the following conversation history into a high-density, lossless Context Checkpoint.
Capture:
1. Primary Goal & User Directives
2. Key Architectural Decisions & Discoveries
3. Files Created, Modified, or Inspected
4. Tools Executed and Their Key Outputs
5. Current State & Pending Next Steps

History to summarize:
%s

Format your response within <CONTEXT_SUMMARY>...</CONTEXT_SUMMARY> tags. Be concise, technical, and precise.`, historyText.String())

	// 3. Request summary from provider
	req := &provider.CompletionRequest{
		SystemPrompt:   "You are an expert context compaction engine.",
		Messages:       []provider.Message{{Role: provider.RoleUser, Content: compactPrompt}},
		Model:          s.Config.Model,
		MaxTokens:      4096,
		ThinkingBudget: 2048,
		Temperature:    0.3,
	}

	resp, err := s.Provider.Stream(ctx, req, func(ev provider.StreamEvent) error { return nil })
	if err != nil {
		return nil, fmt.Errorf("compaction failed: %w", err)
	}

	summary := strings.TrimSpace(resp.Content)

	// 4. Retain only the summary + last 2 recent user/assistant turns
	recentCount := 2
	if len(s.History) < recentCount {
		recentCount = len(s.History)
	}
	recentMessages := s.History[len(s.History)-recentCount:]

	newHistory := []provider.Message{
		{
			Role:    provider.RoleUser,
			Content: fmt.Sprintf("[Compacted Context Checkpoint]\n%s", summary),
		},
		{
			Role:    provider.RoleAssistant,
			Content: "Understood. I have loaded the compacted context checkpoint and retain full memory of our previous progress. Let's continue.",
		},
	}
	newHistory = append(newHistory, recentMessages...)
	s.History = newHistory

	afterTokens := s.GetContextUsage().TotalUsed
	freed := beforeTokens - afterTokens
	if freed < 0 {
		freed = 0
	}
	pctFreed := 0.0
	if beforeTokens > 0 {
		pctFreed = (float64(freed) / float64(beforeTokens)) * 100.0
	}

	return &CompactionResult{
		OriginalTokens: beforeTokens,
		CompactTokens:  afterTokens,
		FreedTokens:    freed,
		PercentFreed:   pctFreed,
		Summary:        summary,
		SnapshotFile:   snapFile,
	}, nil
}
