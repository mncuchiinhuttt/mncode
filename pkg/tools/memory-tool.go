package tools

import (
	"context"
	"fmt"
	"strings"

	"mncode/pkg/memory"
)

// MemoryRememberTool allows agents to save new rules, conventions, and self-reflections into shared memory.
type MemoryRememberTool struct {
	WorkspaceDir string
}

func (t *MemoryRememberTool) Name() string {
	return "memory_remember"
}

func (t *MemoryRememberTool) Description() string {
	return "Store an insight, architectural decision, repository convention, or bug lesson into shared memory across chat sessions."
}

func (t *MemoryRememberTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Topic": map[string]interface{}{
				"type":        "string",
				"description": "Short normalized topic slug (e.g. 'auth-token', 'go-test-timeout')",
			},
			"Category": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"convention", "architecture", "gotcha_bug", "user_preference", "toolchain"},
				"description": "Classification of the memory",
			},
			"Summary": map[string]interface{}{
				"type":        "string",
				"description": "Concise takeaway or rule (e.g. 'Always use -count=1 flag on go test')",
			},
			"Mistake": map[string]interface{}{
				"type":        "string",
				"description": "Optional: what was attempted and failed",
			},
			"Correction": map[string]interface{}{
				"type":        "string",
				"description": "Optional: the verified working approach",
			},
			"Tier": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"workspace", "global"},
				"description": "Sharing scope: 'workspace' (shared across all sessions in repo) or 'global' (cross-project)",
			},
		},
		"required": []string{"Topic", "Summary"},
	}
}

func (t *MemoryRememberTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	topic, _ := args["Topic"].(string)
	summary, _ := args["Summary"].(string)
	if topic == "" || summary == "" {
		return "", fmt.Errorf("both 'Topic' and 'Summary' are required")
	}

	catStr, _ := args["Category"].(string)
	category := memory.CategoryConvention
	if catStr != "" {
		category = memory.MemoryCategory(catStr)
	}

	tierStr, _ := args["Tier"].(string)
	tier := memory.TierWorkspace
	if tierStr == "global" {
		tier = memory.TierGlobal
	}

	mistake, _ := args["Mistake"].(string)
	correction, _ := args["Correction"].(string)

	store, err := memory.NewHierarchicalStore(t.WorkspaceDir)
	if err != nil {
		return "", fmt.Errorf("initialize memory store: %w", err)
	}

	lesson := memory.ReflectiveLesson{
		Topic:      topic,
		Category:   category,
		Summary:    summary,
		Mistake:    mistake,
		Correction: correction,
		Confidence: 5,
		Source:     "agent-remember",
	}

	item, updated, err := memory.EvolveMemory(store, lesson, tier)
	if err != nil {
		return "", fmt.Errorf("save memory: %w", err)
	}

	action := "Created new"
	if updated {
		action = "Evolved & updated"
	}
	return fmt.Sprintf("%s %s memory entry for topic %q (ID: %s).", action, tier, item.Topic, item.ID), nil
}

// MemoryRecallTool queries shared memories.
type MemoryRecallTool struct {
	WorkspaceDir string
}

func (t *MemoryRecallTool) Name() string {
	return "memory_recall"
}

func (t *MemoryRecallTool) Description() string {
	return "Query shared repository and global memories for relevant conventions, past architectural decisions, or bug fixes."
}

func (t *MemoryRecallTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Query": map[string]interface{}{
				"type":        "string",
				"description": "Topic or keyword query to search for",
			},
		},
		"required": []string{"Query"},
	}
}

func (t *MemoryRecallTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, _ := args["Query"].(string)
	store, err := memory.NewHierarchicalStore(t.WorkspaceDir)
	if err != nil {
		return "", fmt.Errorf("initialize memory store: %w", err)
	}

	wsItems, glItems := memory.GetRelevantMemories(store, query, nil, 10)
	if len(wsItems) == 0 && len(glItems) == 0 {
		return fmt.Sprintf("No shared memories found matching query %q.", query), nil
	}

	var sb strings.Builder
	if len(wsItems) > 0 {
		sb.WriteString("=== Shared Workspace Memories ===\n")
		for _, it := range wsItems {
			sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n", strings.ToUpper(string(it.Category)), it.Topic, it.Summary))
			if it.Correction != "" {
				sb.WriteString(fmt.Sprintf("  Correction: %s\n", it.Correction))
			}
		}
	}
	if len(glItems) > 0 {
		sb.WriteString("\n=== Global User Memories ===\n")
		for _, it := range glItems {
			sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n", strings.ToUpper(string(it.Category)), it.Topic, it.Summary))
		}
	}
	return sb.String(), nil
}
