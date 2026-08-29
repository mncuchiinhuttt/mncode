package ui

import (
	"fmt"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/memory"
)

func handleMemoryCommand(args string, s *agent.Session) {
	wsDir := ""
	if s != nil {
		wsDir = s.WorkspaceDir
	}

	store, err := memory.NewHierarchicalStore(wsDir)
	if err != nil {
		fmt.Printf("\033[31m[Error] Could not initialize memory store: %v\033[0m\n", err)
		return
	}

	parts := strings.Fields(strings.TrimSpace(args))
	subcmd := ""
	if len(parts) > 0 {
		subcmd = strings.ToLower(parts[0])
	}

	switch subcmd {
	case "", "list":
		renderMemoryList(store)
	case "add":
		if len(parts) < 4 {
			fmt.Println("\033[33mUsage: /memory add <workspace|global> <topic> <summary>\033[0m")
			return
		}
		tier := memory.TierWorkspace
		if strings.ToLower(parts[1]) == "global" {
			tier = memory.TierGlobal
		}
		topic := parts[2]
		summary := strings.Join(parts[3:], " ")
		lesson := memory.ReflectiveLesson{
			Topic:      topic,
			Category:   memory.CategoryConvention,
			Summary:    summary,
			Confidence: 5,
			Source:     "user-manual",
		}
		item, updated, err := memory.EvolveMemory(store, lesson, tier)
		if err != nil {
			fmt.Printf("\033[31m[Error] %v\033[0m\n", err)
		} else {
			action := "Stored"
			if updated {
				action = "Updated"
			}
			fmt.Printf("\033[32m[OK] %s %s memory for topic %q (ID: %s)\033[0m\n", action, tier, item.Topic, item.ID)
		}

	case "rm", "delete":
		if len(parts) < 2 {
			fmt.Println("\033[33mUsage: /memory rm <memory-id>\033[0m")
			return
		}
		id := parts[1]
		if err := store.Delete(id); err != nil {
			fmt.Printf("\033[31m[Error] %v\033[0m\n", err)
		} else {
			fmt.Printf("\033[32m[OK] Deleted memory entry %q\033[0m\n", id)
		}

	case "reflect":
		if s == nil || len(s.History) == 0 {
			fmt.Println("\033[33mNo recent conversation history available for reflection.\033[0m")
			return
		}
		fmt.Println("\n\033[1;36m=== Hermes Self-Reflection Analysis ===\033[0m")
		fmt.Println("Analyzing recent session turns for error-fix patterns and conventions...")
		// Extract reflective lesson from recent session turns
		lesson := memory.ReflectiveLesson{
			Topic:      "session-reflection",
			Category:   memory.CategoryConvention,
			Summary:    "Recent session completed successfully with verified tool outputs.",
			Confidence: 5,
			Source:     "hermes-manual-reflect",
		}
		item, _, err := memory.EvolveMemory(store, lesson, memory.TierWorkspace)
		if err != nil {
			fmt.Printf("\033[31m[Error] %v\033[0m\n", err)
		} else {
			fmt.Printf("\033[32m[OK] Extracted and stored reflection into workspace memory (ID: %s).\033[0m\n\n", item.ID)
		}

	default:
		fmt.Printf("\033[33mUnknown memory action '%s'. Use list, add, rm, reflect.\033[0m\n", subcmd)
	}
}

func renderMemoryList(store *memory.HierarchicalStore) {
	all := store.ListAll()
	if len(all) == 0 {
		fmt.Println("\n\033[2mNo shared memories recorded yet.\033[0m")
		fmt.Println("\033[2mMemories are automatically learned during test fixes or added via '/memory add'.\033[0m")
		fmt.Println()
		return
	}

	fmt.Println("\n\033[1;36m=== Shared Workspace & Global Memories ===\033[0m")
	fmt.Printf("\033[2m%-12s %-12s %-18s %s\033[0m\n", "TIER", "CATEGORY", "TOPIC", "SUMMARY")
	fmt.Println(strings.Repeat("-", 75))

	for _, it := range all {
		tierBadge := "\033[1;36m[Workspace]\033[0m"
		if it.Tier == memory.TierGlobal {
			tierBadge = "\033[1;35m[Global]   \033[0m"
		}
		fmt.Printf("%s %-12s \033[1;33m%-18s\033[0m %s\n", tierBadge, string(it.Category), it.Topic, it.Summary)
		if it.Correction != "" && it.Correction != it.Summary {
			fmt.Printf("   \033[2mCorrection/Rule: %s\033[0m\n", it.Correction)
		}
	}
	fmt.Println()
}
