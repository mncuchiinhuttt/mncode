package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
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
	} else if strings.HasPrefix("/model", cmd) && len(cmd) >= 3 {
		cmd = "/model"
	} else if strings.HasPrefix("/remote", cmd) && len(cmd) >= 3 {
		cmd = "/remote"
	} else if strings.HasPrefix("/benchmark", cmd) && len(cmd) >= 3 {
		cmd = "/benchmark"
	} else if strings.HasPrefix("/security", cmd) && len(cmd) >= 3 {
		cmd = "/security"
	} else if strings.HasPrefix("/seed", cmd) && len(cmd) >= 3 {
		cmd = "/seed"
	} else if strings.HasPrefix("/search", cmd) && len(cmd) >= 3 {
		cmd = "/search"
	} else if strings.HasPrefix("/export", cmd) && len(cmd) >= 3 {
		cmd = "/export"
	} else if strings.HasPrefix("/combo", cmd) && len(cmd) >= 3 {
		cmd = "/combo"
	} else if strings.HasPrefix("/debate", cmd) && len(cmd) >= 3 {
		cmd = "/debate"
	} else if strings.HasPrefix("/service", cmd) && len(cmd) >= 3 {
		cmd = "/service"
	} else if strings.HasPrefix("/budget", cmd) && len(cmd) >= 3 {
		cmd = "/budget"
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
		ShowSessionStatus(s)
	case "/config", "/settings", "/cfg":
		HandleConfigCommand(parts, s)
	case "/context", "/ctx":
		HandleContextCommand(parts, s)
	case "/compact", "/compress":
		HandleCompactCommand(s)
	case "/sync", "/telemetry":
		HandleSyncCommand(parts, s)
	case "/feedback":
		HandleFeedbackCommand(parts, s)
	case "/usage", "/tokens", "/cost", "/stats":
		ShowUsageStats(s)
	case "/quota", "/limits":
		ShowQuotaDashboard(s)
	case "/search", "/search-setup":
		HandleSearchCommand(parts, s)
	case "/account", "/accounts":
		HandleAccountCommand(parts, s)
	case "/logout":
		if len(parts) > 1 {
			HandleAccountCommand([]string{"/account", "remove", parts[1]}, s)
		} else {
			HandleMncodeLogoutCommand(s)
		}
	case "/login":
		if s.Config.GetTelemetryKey() == "" {
			HandleMncodeLoginCommand(parts, s)
		} else {
			fmt.Printf("\n%s You are already logged in and active in this session! (Use %s to inspect session or %s for AI providers)\n\n", BoldPastelPink("[mncode]"), BoldCyan("/status"), BoldCyan("/account"))
		}
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
		s.Config.SetSetting("troll_mode", newVal)
		_ = config.SaveConfig(s.Config)
		if newVal == "true" {
			fmt.Printf("\n%s Brainrot & Troll Mode enabled! [THINK][MAX] (Full Gen Z / Sigma Dev + Harmless Troll Pranks)\n\n", BoldPastelPink("[PRO MAX]"))
		} else {
			fmt.Printf("\n%s Brainrot & Troll Mode disabled. [PRO] (Standard Professional Dev)\n\n", BoldCyan("[Config]"))
		}
	case "/diff":
		HandleDiffCommand(parts, s)
	case "/steer":
		HandleSteerCommand(parts, s)
	case "/queue":
		HandleQueueCommand(parts, s)
	case "/troll":
		curTroll := s.Config.GetSetting("troll_mode", "false")
		newTroll := "true"
		if len(parts) > 1 {
			if strings.ToLower(parts[1]) == "off" || strings.ToLower(parts[1]) == "false" {
				newTroll = "false"
			}
		} else if curTroll == "true" {
			newTroll = "false"
		}
		s.Config.SetSetting("troll_mode", newTroll)
		_ = config.SaveConfig(s.Config)
		if newTroll == "true" {
			fmt.Printf("\n%s Troll Mode ENABLED! [SIGMA] (Occasional harmless scare pranks before tools)\n\n", BoldPastelPink("[Troll Mode]"))
		} else {
			fmt.Printf("\n%s Troll Mode disabled. [PRO] (Strict serious dev mode)\n\n", BoldCyan("[Troll Mode]"))
		}
	case "/scan":
		HandleScanCommand(parts, s)
	case "/plan":
		HandlePlanCommand(parts, s)
	case "/mcp":
		HandleMCPCommand(parts, s)
	case "/research", "/deepresearch", "/deep-research":
		HandleResearchCommand(parts, s)
	case "/litrev", "/lit-review":
		HandleLitRevCommand(parts, s)
	case "/recap", "/summary":
		HandleRecapCommand(parts, s)
	case "/update", "/upgrade":
		HandleUpdateCommand(parts, s)
	case "/version", "/v":
		ShowVersionInfo()
	case "/undo":
		HandleUndoCommand(parts, s)
	case "/rewind":
		HandleRewindCommand(parts, s)
	case "/checkpoint":
		HandleCheckpointCommand(parts, s)
	case "/commit":
		HandleCommitCommand(parts, s)
	case "/test":
		HandleTestCommand(parts, s)
	case "/heal":
		HandleTestCommand([]string{"/test", "--heal"}, s)
	case "/review":
		HandleReviewCommand(parts, s)
	case "/share":
		HandleShareCommand(parts, s)
	case "/export", "/trajectory", "/sharegpt", "/export-training":
		HandleTrajectoryExportCommand(parts, s)
	case "/pr":
		HandlePRCommand(parts, s)
	case "/doctor":
		HandleDoctorCommand(parts, s)
	case "/symbol", "/find-symbol":
		HandleSymbolCommand(parts, s)
	case "/scratch":
		HandleScratchCommand(parts, s)
	case "/resolve", "/conflict", "/conflicts":
		HandleResolveCommand(parts, s)
	case "/db", "/sql", "/database":
		HandleDBCommand(parts, s)
	case "/api", "/curl", "/http":
		HandleAPICommand(parts)
	case "/tree", "/files":
		HandleTreeCommand(parts, s)
	case "/changelog", "/release-notes":
		HandleChangelogCommand(parts, s)
	case "/resume", "/history":
		HandleResumeCommand(parts, s)
	case "/remote", "/companion", "/mobile":
		HandleRemoteCommand(parts, s)
	case "/benchmark", "/bench", "/perf":
		HandleBenchmarkCommand(parts, s)
	case "/security", "/audit", "/vuln":
		HandleSecurityCommand(parts, s)
	case "/seed", "/mock", "/mockdata":
		HandleSeedCommand(parts, s)
	case "/combo", "/combos", "/swarm", "/pipeline":
		args := ""
		if len(parts) > 1 {
			args = strings.Join(parts[1:], " ")
		}
		handleComboCommand(args, s)
	case "/debate", "/arena":
		args := ""
		if len(parts) > 1 {
			args = strings.Join(parts[1:], " ")
		}
		handleDebateCommand(args, s)
	case "/service", "/svc", "/daemon":
		args := ""
		if len(parts) > 1 {
			args = strings.Join(parts[1:], " ")
		}
		handleServiceCommand(args, s)
	case "/budget", "/limit":
		args := ""
		if len(parts) > 1 {
			args = strings.Join(parts[1:], " ")
		}
		handleBudgetCommand(args, s)
	default:
		ShowSlashPalette()
	}
	return true
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
