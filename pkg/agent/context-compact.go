package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"mncode/pkg/persistence"
	"mncode/pkg/provider"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

var historyMu sync.Mutex

// ensureSessionIdentity gives every history writer the same durable identity.
// Existing IDs are never replaced; generated IDs are stable for this Session.
func ensureSessionIdentity(s *Session) string {
	historyMu.Lock()
	defer historyMu.Unlock()
	if s.ID == "" || s.ID == "mncode-main" {
		s.ID = fmt.Sprintf("session-%d", time.Now().UTC().UnixNano())
	}
	return s.ID
}

// appendHistory is the single mutation boundary used by all autonomous loops.
func appendHistory(s *Session, msg provider.Message) {
	historyMu.Lock()
	defer historyMu.Unlock()
	s.History = append(s.History, cloneMessage(msg))
}

func cloneMessage(msg provider.Message) provider.Message {
	msg.Images = append([]provider.ImageData(nil), msg.Images...)
	msg.ToolCalls = append([]provider.ToolCall(nil), msg.ToolCalls...)
	for i := range msg.ToolCalls {
		if msg.ToolCalls[i].Arguments != nil {
			msg.ToolCalls[i].Arguments = make(map[string]interface{}, len(msg.ToolCalls[i].Arguments))
			for key, value := range msg.ToolCalls[i].Arguments {
				msg.ToolCalls[i].Arguments[key] = value
			}
		}
	}
	msg.ToolResults = append([]provider.ToolResult(nil), msg.ToolResults...)
	return msg
}

func historySnapshot(s *Session) []provider.Message {
	historyMu.Lock()
	defer historyMu.Unlock()
	return append([]provider.Message(nil), s.History...)
}

// CompactHistory summarizes history before replacing it. Every failure occurs
// before the in-memory history is changed, so callers can safely retry.
func (s *Session) CompactHistory(ctx context.Context) (*CompactionResult, error) {
	historyMu.Lock()
	history := append([]provider.Message(nil), s.History...)
	historyMu.Unlock()
	if len(history) <= 2 {
		return nil, fmt.Errorf("conversation history too short to compact (%d messages)", len(history))
	}
	if s.Provider == nil {
		return nil, fmt.Errorf("compaction provider is unavailable")
	}

	beforeTokens := s.GetContextUsage().TotalUsed
	snapFile, err := writeHistorySnapshot(history)
	if err != nil {
		return nil, fmt.Errorf("compaction snapshot failed: %w", err)
	}

	var historyText strings.Builder
	for idx, msg := range history {
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
			preview := tr.Content
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			historyText.WriteString(fmt.Sprintf("[Tool Result %s: %s]\n", tr.Name, preview))
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

	req := &provider.CompletionRequest{
		SystemPrompt: "You are an expert context compaction engine.",
		Messages:     []provider.Message{{Role: provider.RoleUser, Content: compactPrompt}},
		Model:        s.Config.Model, MaxTokens: 4096, ThinkingBudget: 2048, Temperature: 0.3,
	}
	resp, err := s.streamProvider(ctx, req, func(provider.StreamEvent) error { return nil })
	if err != nil {
		return nil, fmt.Errorf("compaction summary failed: %w", err)
	}
	summary := strings.TrimSpace(resp.Content)
	if summary == "" {
		return nil, fmt.Errorf("compaction summary failed: provider returned empty summary")
	}

	newHistory, err := boundedCompactedHistory(history, summary, s.Config.GetContextWindowTokens())
	if err != nil {
		return nil, err
	}
	if err := persistCompactedHistory(ctx, s, newHistory); err != nil {
		return nil, fmt.Errorf("compaction store failed: %w", err)
	}
	historyMu.Lock()
	s.History = newHistory
	historyMu.Unlock()
	s.InvalidatePromptCache("compaction")

	afterTokens := s.GetContextUsage().TotalUsed
	freed := beforeTokens - afterTokens
	if freed < 0 {
		freed = 0
	}
	pctFreed := 0.0
	if beforeTokens > 0 {
		pctFreed = float64(freed) / float64(beforeTokens) * 100
	}
	return &CompactionResult{OriginalTokens: beforeTokens, CompactTokens: afterTokens, FreedTokens: freed, PercentFreed: pctFreed, Summary: summary, SnapshotFile: snapFile}, nil
}

func writeHistorySnapshot(history []provider.Message) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".mncode", "snapshots")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("snapshot-%d.json", time.Now().UTC().UnixNano()))
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".snapshot-*.tmp")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := replaceExistingFile(tmpName, path); err != nil {
		return "", err
	}
	return path, nil
}

type historyUnit struct {
	index    int
	messages []provider.Message
	tokens   int
}

func boundedCompactedHistory(history []provider.Message, summary string, limit int) ([]provider.Message, error) {
	if limit <= 0 {
		limit = 1000000
	}
	checkpoint := provider.Message{Role: provider.RoleUser, Content: "[Compacted Context Checkpoint]\n" + summary}
	ack := provider.Message{Role: provider.RoleAssistant, Content: "Understood. I have loaded the compacted context checkpoint and retain the previous progress. Let's continue."}
	budget := limit - messageTokens(checkpoint) - messageTokens(ack)
	if budget <= 0 {
		return nil, fmt.Errorf("compaction token budget exhausted")
	}
	units := compactUnits(history)
	selected := make([]historyUnit, 0, len(units))
	used := 0
	// Preserve the first user intent (and its tool group, if any).
	if len(units) > 0 {
		selected = append(selected, units[0])
		used += units[0].tokens
	}
	tail := make([]historyUnit, 0, len(units))
	for i := len(units) - 1; i >= 0; i-- {
		if len(selected) > 0 && sameUnit(selected[0], units[i]) {
			continue
		}
		if used+units[i].tokens > budget {
			continue
		}
		tail = append(tail, units[i])
		used += units[i].tokens
	}
	if len(selected) == 0 || used > budget {
		return nil, fmt.Errorf("compaction token budget exhausted")
	}
	// Restore tail order while retaining the first intent at the front.
	for i, j := 0, len(tail)-1; i < j; i, j = i+1, j-1 {
		tail[i], tail[j] = tail[j], tail[i]
	}
	selected = append(selected, tail...)
	out := []provider.Message{checkpoint, ack}
	for _, unit := range selected {
		out = append(out, unit.messages...)
	}
	return out, nil
}

func compactUnits(history []provider.Message) []historyUnit {
	units := make([]historyUnit, 0, len(history))
	for i := 0; i < len(history); i++ {
		msg := history[i]
		unit := historyUnit{index: i, messages: []provider.Message{msg}, tokens: messageTokens(msg)}
		if len(msg.ToolCalls) > 0 {
			for i+1 < len(history) && history[i+1].Role == provider.RoleTool {
				unit.messages = append(unit.messages, history[i+1])
				unit.tokens += messageTokens(history[i+1])
				i++
			}
		}
		units = append(units, unit)
	}
	return units
}

func sameUnit(a, b historyUnit) bool {
	return len(a.messages) > 0 && len(b.messages) > 0 && a.index == b.index
}
func messageTokens(msg provider.Message) int {
	b, _ := json.Marshal(msg)
	n := len(b) / 4
	if n < 1 {
		n = 1
	}
	return n
}

func persistCompactedHistory(ctx context.Context, s *Session, history []provider.Message) error {
	id := ensureSessionIdentity(s)
	store, err := openCanonicalStore()
	if err != nil {
		return err
	}
	defer store.Close()
	now := time.Now().UTC()
	model, providerName := "", ""
	if s.Config != nil {
		model, providerName = s.Config.Model, string(s.Config.Provider)
	}
	title, turns := "New Session", 0
	for _, msg := range history {
		if msg.Role != provider.RoleUser {
			continue
		}
		turns++
		if title == "New Session" && strings.TrimSpace(msg.Content) != "" {
			title = strings.TrimSpace(msg.Content)
			if len([]rune(title)) > 45 {
				title = string([]rune(title)[:42]) + "..."
			}
		}
	}
	return store.SaveSession(ctx, persistence.SessionRecord{
		ID: id, ChatID: id, WorkspaceDir: s.WorkspaceDir, Title: title,
		Model: model, Provider: providerName, Turns: turns, UpdatedAt: now,
		CreatedAt: now, Messages: canonicalMessages(id, history, now),
	})
}
