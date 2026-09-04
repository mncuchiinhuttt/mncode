package memory

import (
	"fmt"
	"mncode/pkg/artifacts"
	"sort"
	"strings"
)

// GetRelevantMemories queries and scores shared workspace memories and global memories
// based on topic overlap with user prompt keywords and file paths.
func GetRelevantMemories(store *HierarchicalStore, userPrompt string, files []string, limit int) (workspace []MemoryItem, global []MemoryItem) {
	if store == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	all := store.ListAll()
	if len(all) == 0 {
		return nil, nil
	}

	promptLower := strings.ToLower(userPrompt)
	fileTokens := make(map[string]bool)
	for _, f := range files {
		for _, seg := range strings.Split(f, "/") {
			if len(seg) > 2 {
				fileTokens[strings.ToLower(seg)] = true
			}
		}
	}

	type scoredItem struct {
		item  MemoryItem
		score int
	}

	var scoredWorkspace []scoredItem
	var scoredGlobal []scoredItem

	for _, it := range all {
		score := it.Confidence
		topicLower := strings.ToLower(it.Topic)

		if strings.Contains(promptLower, topicLower) {
			score += 10
		}
		if fileTokens[topicLower] {
			score += 5
		}
		for tok := range fileTokens {
			if strings.Contains(strings.ToLower(it.Summary), tok) {
				score += 3
			}
		}

		entry := scoredItem{item: it, score: score}
		if it.Tier == TierWorkspace {
			scoredWorkspace = append(scoredWorkspace, entry)
		} else {
			scoredGlobal = append(scoredGlobal, entry)
		}
	}

	sortByScore := func(slice []scoredItem) []MemoryItem {
		sort.Slice(slice, func(i, j int) bool {
			return slice[i].score > slice[j].score
		})
		var result []MemoryItem
		for i := 0; i < len(slice) && i < limit; i++ {
			result = append(result, slice[i].item)
		}
		return result
	}

	return sortByScore(scoredWorkspace), sortByScore(scoredGlobal)
}

// FormatPromptMemoryContext formats shared memories into clean XML tags for prompt injection.
func FormatPromptMemoryContext(workspaceItems, globalItems []MemoryItem) string {
	var sb strings.Builder

	if len(workspaceItems) > 0 {
		sb.WriteString("<shared-workspace-memories>\n")
		for _, it := range workspaceItems {
			line := fmt.Sprintf("- [%s] %s: %s", strings.ToUpper(string(it.Category)), it.Topic, it.Summary)
			if it.Correction != "" && it.Correction != it.Summary {
				line += fmt.Sprintf(" | Rule: %s", it.Correction)
			}
			sb.WriteString(artifacts.ScrubSecrets(line) + "\n")
		}
		sb.WriteString("</shared-workspace-memories>\n\n")
	}

	if len(globalItems) > 0 {
		sb.WriteString("<global-user-memories>\n")
		for _, it := range globalItems {
			line := fmt.Sprintf("- [%s] %s: %s", strings.ToUpper(string(it.Category)), it.Topic, it.Summary)
			sb.WriteString(artifacts.ScrubSecrets(line) + "\n")
		}
		sb.WriteString("</global-user-memories>\n\n")
	}

	return sb.String()
}
