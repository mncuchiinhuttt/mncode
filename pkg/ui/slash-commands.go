package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"mncode/pkg/stats"
	"sort"
	"strings"
)

// HandleSlashCommand processes REPL slash commands. Returns true if handled.
func HandleSlashCommand(input string, s *agent.Session) bool {
	if !strings.HasPrefix(input, "/") {
		return false
	}

	trimmed := strings.TrimSpace(input)
	if trimmed == "/" || trimmed == "/menu" {
		OpenInteractiveSlashMenu(s)
		return true
	}

	// Resolve /1, /2, etc.
	trimmed = ResolveNumericSlashCommand(trimmed)

	parts := strings.Fields(trimmed)
	cmd := strings.ToLower(parts[0])

	// Prefix / shortcut resolution
	if strings.HasPrefix("/skills", cmd) && len(cmd) >= 3 {
		cmd = "/skills"
	} else if strings.HasPrefix("/agents", cmd) && len(cmd) >= 3 {
		cmd = "/agents"
	} else if strings.HasPrefix("/rules", cmd) && len(cmd) >= 3 {
		cmd = "/rules"
	} else if strings.HasPrefix("/status", cmd) && len(cmd) >= 3 {
		cmd = "/status"
	} else if strings.HasPrefix("/clear", cmd) && len(cmd) >= 3 {
		cmd = "/clear"
	} else if strings.HasPrefix("/account", cmd) && len(cmd) >= 3 {
		cmd = "/account"
	} else if strings.HasPrefix("/login", cmd) && len(cmd) >= 3 {
		cmd = "/login"
	} else if strings.HasPrefix("/model", cmd) && len(cmd) >= 3 {
		cmd = "/model"
	}

	switch cmd {
	case "/help", "/?":
		ShowSlashPalette()
	case "/skills", "/skill":
		HandleSkillCommand(parts, s)
	case "/agents", "/subagents", "/workflows":
		OpenSubagentMonitorView(s)
	case "/rules":
		showRules(s)
	case "/model", "/models":
		if len(parts) > 1 {
			targetModel := parts[1]
			s.Config.Model = targetModel
			for _, m := range curatedModels {
				if strings.EqualFold(m.ID, targetModel) && m.Provider != "" {
					s.Config.Provider = m.Provider
					break
				}
			}
			_ = config.SaveConfig(s.Config)
			_ = s.EnsureProvider()
			RenderStickyHeader(s)
			fmt.Printf("Model updated to: %s (Provider: %s)\n", BoldCyan(s.Config.Model), s.Config.Provider)
		} else {
			OpenInteractiveModelSelector(s)
		}
	case "/effort", "/think":
		HandleEffortCommand(parts, s)
	case "/goal":
		HandleGoalCommand(parts, s)
	case "/workflow", "/mode":
		HandleWorkflowCommand(parts, s)
	case "/theme", "/themes":
		OpenInteractiveThemeSelector(s)
	case "/clear":
		s.History = nil
		fmt.Print("\033[H\033[2J") // Clear terminal
		fmt.Println("Session history cleared.")
	case "/status":
		showStatus(s)
	case "/config", "/settings", "/cfg":
		HandleConfigCommand(parts, s)
	case "/context", "/ctx":
		ShowContextUsage(s)
	case "/compact", "/compress":
		HandleCompactCommand(s)
	case "/usage", "/tokens", "/cost", "/stats":
		ShowUsageStats(s)
	case "/quota", "/limits":
		ShowQuotaDashboard(s)
	case "/account", "/accounts":
		HandleAccountCommand(parts, s)
	case "/logout":
		if len(parts) > 1 {
			HandleAccountCommand([]string{"/account", "remove", parts[1]}, s)
		} else {
			OpenInteractiveAccountMenu(s)
		}
	case "/login":
		HandleLoginPrompt(parts, s)
	case "/btw":
		HandleBTWCommand(parts, s)
	case "/brainrot":
		cur := s.Config.GetSetting("brainrot_mode", "false")
		newVal := "true"
		if len(parts) > 1 {
			if strings.ToLower(parts[1]) == "off" || strings.ToLower(parts[1]) == "false" {
				newVal = "false"
			}
		} else if cur == "true" {
			newVal = "false"
		}
		s.Config.SetSetting("brainrot_mode", newVal)
		_ = config.SaveConfig(s.Config)
		if newVal == "true" {
			fmt.Printf("\n%s Brainrot Mode enabled! 🧠🔥 (Full Gen Z / Sigma Dev Persona)\n\n", BoldPastelPink("[PRO MAX]"))
		} else {
			fmt.Printf("\n%s Brainrot Mode disabled. 💼 (Standard Professional Dev)\n\n", BoldCyan("[Config]"))
		}
	case "/resume", "/history":
		HandleResumeCommand(parts, s)
	default:
		fmt.Printf("Unknown command '%s'.\n", cmd)
		ShowSlashPalette()
	}
	return true
}

func showSkills(s *agent.Session) {
	if s.Catalog == nil || len(s.Catalog.Skills) == 0 {
		fmt.Println("No skills loaded.")
		return
	}
	var names []string
	for name := range s.Catalog.Skills {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Printf("\n%s (%d skills loaded):\n", BoldCyan("Claude Skills"), len(names))
	for _, n := range names {
		sk := s.Catalog.Skills[n]
		fmt.Printf("  • %-24s %s\n", Bold(sk.Name), GrayText(sk.Description))
	}
	fmt.Println()
}

func showAgents(s *agent.Session) {
	if s.Catalog == nil || len(s.Catalog.Agents) == 0 {
		fmt.Println("No agents loaded.")
		return
	}
	fmt.Printf("\n%s (%d agents):\n", BoldMagenta("mnCode Specialized Agents"), len(s.Catalog.Agents))
	for name, a := range s.Catalog.Agents {
		fmt.Printf("  • %-20s (%s)\n", Bold(name), a.Role)
	}
	fmt.Println()
}

func showRules(s *agent.Session) {
	if s.Catalog == nil || len(s.Catalog.Rules) == 0 {
		fmt.Println("No rules loaded.")
		return
	}
	fmt.Printf("\n%s (%d rules):\n", BoldYellow("Workspace Rules"), len(s.Catalog.Rules))
	for name := range s.Catalog.Rules {
		fmt.Printf("  • %s\n", name)
	}
	fmt.Println()
}

func showStatus(s *agent.Session) {
	fmt.Printf("\n%s\n", BoldCyan("Session Status:"))
	fmt.Printf("  Workspace:    %s\n", s.WorkspaceDir)
	fmt.Printf("  Provider:     %s\n", s.Config.Provider)
	fmt.Printf("  Model:        %s\n", s.Config.Model)
	fmt.Printf("  Effort Level: %s (%d tokens budget)\n", BoldGreen(strings.ToUpper(s.Config.Effort)), s.Config.ThinkingBudget)
	fmt.Printf("  Workflow:     %s\n", BoldCyan(strings.ToUpper(s.Config.Workflow)))
	fmt.Printf("  Auto-Approve: %v\n", s.Config.AutoApprove)
	fmt.Printf("  History Size: %d messages\n", len(s.History))
	if tracker, ok := s.Tracker.(*stats.Tracker); ok && tracker != nil {
		today := tracker.GetToday()
		lifetime := tracker.GetLifetime()
		fmt.Printf("  Today Tokens: %s (%d requests)\n", BoldGreen(formatNumber(today.TotalTokens)), today.Requests)
		fmt.Printf("  Total Tokens: %s (%d requests)\n", BoldYellow(formatNumber(lifetime.TotalTokens)), lifetime.Requests)
	}
	fmt.Println()
}
