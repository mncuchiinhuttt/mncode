package agent

import (
	"testing"

	"mncode/pkg/provider"
)

func TestBoundedCompactedHistoryPreservesToolGroups(t *testing.T) {
	call := provider.ToolCall{ID: "call-1", Name: "inspect", Arguments: map[string]interface{}{"path": "a.go"}}
	history := []provider.Message{
		{Role: provider.RoleUser, Content: "original intent"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{call}},
		{Role: provider.RoleTool, ToolResults: []provider.ToolResult{{ToolCallID: "call-1", Name: "inspect", Content: "result"}}},
		{Role: provider.RoleUser, Content: "latest request"},
		{Role: provider.RoleAssistant, Content: "latest answer"},
	}
	compacted, err := boundedCompactedHistory(history, "summary", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if compacted[0].Role != provider.RoleUser || compacted[1].Role != provider.RoleAssistant {
		t.Fatalf("missing checkpoint pair: %+v", compacted[:2])
	}
	foundCall, foundResult := false, false
	for _, msg := range compacted {
		if len(msg.ToolCalls) > 0 {
			foundCall = true
		}
		if len(msg.ToolResults) > 0 {
			foundResult = true
		}
	}
	if !foundCall || !foundResult {
		t.Fatalf("tool call/result group was split: %+v", compacted)
	}
}

func TestBoundedCompactedHistoryRejectsExhaustedBudget(t *testing.T) {
	_, err := boundedCompactedHistory([]provider.Message{{Role: provider.RoleUser, Content: "intent"}}, "summary", 1)
	if err == nil {
		t.Fatal("expected token budget exhaustion")
	}
}

func TestBoundedCompactedHistoryPreservesParallelToolGroups(t *testing.T) {
	calls := []provider.ToolCall{
		{ID: "call-1", Name: "read", Arguments: map[string]interface{}{"path": "a.go"}},
		{ID: "call-2", Name: "read", Arguments: map[string]interface{}{"path": "b.go"}},
	}
	history := []provider.Message{
		{Role: provider.RoleUser, Content: "read files"},
		{Role: provider.RoleAssistant, ToolCalls: calls},
		{Role: provider.RoleTool, ToolResults: []provider.ToolResult{{ToolCallID: "call-1", Name: "read", Content: "content a"}}},
		{Role: provider.RoleTool, ToolResults: []provider.ToolResult{{ToolCallID: "call-2", Name: "read", Content: "content b"}}},
		{Role: provider.RoleUser, Content: "summarize"},
		{Role: provider.RoleAssistant, Content: "done"},
	}
	compacted, err := boundedCompactedHistory(history, "summary", 2000)
	if err != nil {
		t.Fatal(err)
	}
	resultCount := 0
	for _, msg := range compacted {
		if msg.Role == provider.RoleTool {
			resultCount++
		}
	}
	if resultCount != 2 {
		t.Fatalf("expected 2 tool results in compacted history, got %d", resultCount)
	}
}
