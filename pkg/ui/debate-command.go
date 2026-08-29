package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mncode/pkg/agent"
	"mncode/pkg/combos"
)

func handleDebateCommand(args string, session *agent.Session) {
	if session == nil {
		fmt.Println("\033[31m[Error] Active session required for /debate.\033[0m")
		return
	}

	fmt.Println("\n\033[1;36m=== AI Agent Debate Arena ===\033[0m")

	// ─────────────────────────────────────────────────────────────────────────────
	// Step 1: Ask if user wants to assign explicit roles or run a free peer debate
	// ─────────────────────────────────────────────────────────────────────────────
	fmt.Println("\n\033[1m[Step 1: Role Configuration]\033[0m")
	fmt.Println("  Do you want to assign explicit roles to each agent?")
	fmt.Println("  1) \033[1;32mNo\033[0m  - Free Peer Debate (Default: Open peer discussion & cross-examination)")
	fmt.Println("  2) \033[1;36mYes\033[0m - Structured Role Debate (Roles: Proposer -> Technical Critic -> Judge)")
	fmt.Print("\nChoose style [1-2, default 1]: ")
	styleChoice := strings.TrimSpace(readLineRaw())

	hasRoles := styleChoice == "2" || strings.EqualFold(styleChoice, "yes") || strings.EqualFold(styleChoice, "y")

	// ─────────────────────────────────────────────────────────────────────────────
	// Step 2: Interactive Multi-Select Models (Space to toggle green dot, Enter to confirm)
	// ─────────────────────────────────────────────────────────────────────────────
	fmt.Println()
	selectedModels := selectDebateModelsInteractive()
	if len(selectedModels) < 2 {
		selectedModels = []string{"claude-sonnet-4-6", "deepseek-reasoner"}
	}

	fmt.Printf("\033[1;32m[OK] Selected %d models for debate:\033[0m %s\n", len(selectedModels), strings.Join(selectedModels, ", "))

	// ─────────────────────────────────────────────────────────────────────────────
	// Step 3: Enter the Task / Question to debate
	// ─────────────────────────────────────────────────────────────────────────────
	task := strings.TrimSpace(args)
	if task == "" {
		fmt.Println("\n\033[1m[Step 3: Debate Topic]\033[0m")
		fmt.Print("Enter problem / architecture topic to debate:\n> ")
		task = strings.TrimSpace(readLineRaw())
		if task == "" {
			fmt.Println("\033[33m[Cancelled] No debate topic provided.\033[0m")
			return
		}
	} else {
		fmt.Printf("\n\033[1m[Step 3: Debate Topic]\033[0m %s\n", task)
	}

	// ─────────────────────────────────────────────────────────────────────────────
	// Step 4: Construct and Launch the Dynamic Debate Arena
	// ─────────────────────────────────────────────────────────────────────────────
	now := time.Now()
	var members []combos.ComboMember
	modelProposer := selectedModels[0]
	modelCritic := selectedModels[1]
	modelJudge := selectedModels[0]
	if len(selectedModels) > 2 {
		modelJudge = selectedModels[2]
	}

	if hasRoles {
		members = []combos.ComboMember{
			{
				ID:            "prop-1",
				Role:          "Architect / Proposer",
				BaseAgent:     "planner",
				Model:         modelProposer,
				FallbackModel: "gemini-3.7-flash-high",
				PromptOverlay: "Propose an optimal, concrete technical solution with architecture, data flow, and code.",
			},
			{
				ID:            "crit-1",
				Role:          "Technical Critic (Devil's Advocate)",
				BaseAgent:     "code-reviewer",
				Model:         modelCritic,
				FallbackModel: "gemini-3.7-flash-high",
				PromptOverlay: "Aggressively stress-test the proposal. Find race conditions, single points of failure, scalability bottlenecks, and security flaws.",
			},
			{
				ID:            "judge-1",
				Role:          "Lead Judge & Synthesizer",
				BaseAgent:     "coder",
				Model:         modelJudge,
				FallbackModel: "gemini-3.7-flash-high",
				PromptOverlay: "Review the full debate transcript. Reconcile valid critiques into an authoritative final architecture and implementation deliverable.",
			},
		}
	} else {
		for i, m := range selectedModels {
			members = append(members, combos.ComboMember{
				ID:            fmt.Sprintf("peer-%d", i+1),
				Role:          fmt.Sprintf("Peer Debater %c (%s)", 'A'+rune(i), m),
				BaseAgent:     "coder",
				Model:         m,
				FallbackModel: "gemini-3.7-flash-high",
				PromptOverlay: "Provide your deep engineering perspective and solution for the problem. Critique other models' viewpoints constructively and challenge missing edge cases.",
			})
		}
		members = append(members, combos.ComboMember{
			ID:            "peer-synth",
			Role:          "Consensus Synthesizer",
			BaseAgent:     "coder",
			Model:         modelJudge,
			FallbackModel: "gemini-3.7-flash-high",
			PromptOverlay: "Synthesize the best ideas from all peer perspectives into a unified, actionable deliverable.",
		})
	}

	debateCombo := combos.Combo{
		ID:              fmt.Sprintf("arena-debate-%d", now.UnixNano()),
		Name:            fmt.Sprintf("Debate Arena: %s", strings.Join(selectedModels, " vs ")),
		Description:     fmt.Sprintf("Debate on: %s", task),
		Mode:            combos.ModeDebate,
		MaxDebateRounds: 2,
		Members:         members,
	}

	store, _ := combos.NewStore(session.WorkspaceDir)
	_ = store.Save(debateCombo)
	defer func() { _ = store.Delete(debateCombo.ID) }()

	exec := newSessionComboExecutor(session)
	hud := newDebateArenaHUD(modelProposer, modelCritic, modelJudge)
	runner := combos.NewRunner(store, exec, hud)

	res, err := runner.Run(context.Background(), debateCombo.ID, task)
	if err != nil {
		fmt.Printf("\n\033[31m[Debate Arena Error] %v\033[0m\n\n", err)
		return
	}

	fmt.Println("\n\033[1;32m===========================================================================\033[0m")
	fmt.Println("\033[1;32m                       FINAL SYNTHESIZED VERDICT                           \033[0m")
	fmt.Println("\033[1;32m===========================================================================\033[0m")
	fmt.Println(res.FinalOutput)
	fmt.Println()
}
