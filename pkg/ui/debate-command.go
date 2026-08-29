package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mncode/pkg/agent"
	"mncode/pkg/combos"
)

var popularDebateModels = []struct {
	id   string
	name string
}{
	{"claude-sonnet-4-6", "Claude Sonnet 4.6 (Thinking)"},
	{"deepseek-reasoner", "DeepSeek R1 (Reasoner)"},
	{"gemini-3.7-flash-high", "Gemini 3.7 Flash (High Thinking)"},
	{"o3", "OpenAI o3 (Reasoning Flagship)"},
	{"claude-opus-4-6-thinking", "Claude Opus 4.6 (Flagship Deep Reasoning)"},
	{"gemini-pro-agent", "Gemini 3.1 Pro (Deep Agent Tier)"},
	{"gpt-4.5", "GPT-4.5 Orion"},
	{"gpt-4o", "GPT-4o Omni"},
}

func handleDebateCommand(args string, session *agent.Session) {
	if session == nil {
		fmt.Println("\033[31m[Error] Active session required for /debate.\033[0m")
		return
	}

	fmt.Println("\n\033[1;36m═══ AI Agent Debate Arena Setup ═══\033[0m")

	// ─────────────────────────────────────────────────────────────────────────────
	// Step 1: Ask if user wants to assign explicit roles or run a free peer debate
	// ─────────────────────────────────────────────────────────────────────────────
	fmt.Println("\n\033[1mStep 1: Do you want to assign explicit roles to each agent?\033[0m")
	fmt.Println("  1) \033[1;32mNo\033[0m  - Free Peer Debate (Mặc định: Thảo luận bàn tròn tự do giữa các model)")
	fmt.Println("  2) \033[1;36mYes\033[0m - Structured Role Debate (Phân role: Đề xuất ➔ Phản biện vạch lỗi ➔ Trọng tài)")
	fmt.Print("Choose style [1-2, default 1]: ")
	styleChoice := strings.TrimSpace(readLineRaw())

	hasRoles := styleChoice == "2" || strings.EqualFold(styleChoice, "yes") || strings.EqualFold(styleChoice, "y")

	var modelProposer, modelCritic, modelJudge string
	var freeModels []string

	// ─────────────────────────────────────────────────────────────────────────────
	// Step 2: Choose Agents / Models based on selected style
	// ─────────────────────────────────────────────────────────────────────────────
	fmt.Println("\n\033[1mStep 2: Choose Debater Models\033[0m")
	for i, m := range popularDebateModels {
		fmt.Printf("   %d) %-34s \033[2m(%s)\033[0m\n", i+1, m.name, m.id)
	}

	if hasRoles {
		// Structured Role Selection
		fmt.Print("\nChoose Proposer Model [1-8, default 1: Claude Sonnet 4.6]: ")
		modelProposer = pickDebateModel(readLineRaw(), "claude-sonnet-4-6")

		fmt.Print("Choose Critic Model [1-8, default 2: DeepSeek R1]: ")
		modelCritic = pickDebateModel(readLineRaw(), "deepseek-reasoner")

		fmt.Print("Choose Judge / Decider Model [1-8, default 5: Claude Opus 4.6]: ")
		modelJudge = pickDebateModel(readLineRaw(), "claude-opus-4-6-thinking")
	} else {
		// Free Peer Debate Selection (Select 2 debaters)
		fmt.Print("\nChoose First Debater [1-8, default 1: Claude Sonnet 4.6]: ")
		m1 := pickDebateModel(readLineRaw(), "claude-sonnet-4-6")

		fmt.Print("Choose Second Debater [1-8, default 2: DeepSeek R1]: ")
		m2 := pickDebateModel(readLineRaw(), "deepseek-reasoner")

		freeModels = []string{m1, m2}
		modelProposer = m1
		modelCritic = m2
		modelJudge = m1
	}

	// ─────────────────────────────────────────────────────────────────────────────
	// Step 3: Enter the Task / Question to debate
	// ─────────────────────────────────────────────────────────────────────────────
	task := strings.TrimSpace(args)
	if task == "" {
		fmt.Println("\n\033[1mStep 3: Enter problem / architecture topic to debate:\033[0m")
		fmt.Print("> ")
		task = strings.TrimSpace(readLineRaw())
		if task == "" {
			fmt.Println("\033[33m[Cancelled] No debate topic provided.\033[0m")
			return
		}
	} else {
		fmt.Printf("\n\033[1mStep 3: Debate Topic:\033[0m %s\n", task)
	}

	// ─────────────────────────────────────────────────────────────────────────────
	// Step 4: Construct and Launch the Dynamic Debate Arena
	// ─────────────────────────────────────────────────────────────────────────────
	now := time.Now()
	var members []combos.ComboMember

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
		members = []combos.ComboMember{
			{
				ID:            "peer-1",
				Role:          fmt.Sprintf("Peer Debater A (%s)", freeModels[0]),
				BaseAgent:     "coder",
				Model:         freeModels[0],
				FallbackModel: "gemini-3.7-flash-high",
				PromptOverlay: "Provide your deep perspective and solution for the problem. Critique other models' viewpoints constructively.",
			},
			{
				ID:            "peer-2",
				Role:          fmt.Sprintf("Peer Debater B (%s)", freeModels[1]),
				BaseAgent:     "coder",
				Model:         freeModels[1],
				FallbackModel: "gemini-3.7-flash-high",
				PromptOverlay: "Provide your alternative perspective, challenge assumptions, and highlight trade-offs in the other model's approach.",
			},
			{
				ID:            "peer-synth",
				Role:          "Consensus Synthesizer",
				BaseAgent:     "coder",
				Model:         freeModels[0],
				FallbackModel: "gemini-3.7-flash-high",
				PromptOverlay: "Synthesize the best ideas from both peer perspectives into a unified, actionable deliverable.",
			},
		}
	}

	debateCombo := combos.Combo{
		ID:              fmt.Sprintf("arena-debate-%d", now.UnixNano()),
		Name:            fmt.Sprintf("Debate Arena: %s vs %s", modelProposer, modelCritic),
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

	fmt.Println("\n\033[1;32m═══════════════════════════════════════════════════════════════════════════\033[0m")
	fmt.Println("\033[1;32m                    🏛️  FINAL SYNTHESIZED VERDICT                          \033[0m")
	fmt.Println("\033[1;32m═══════════════════════════════════════════════════════════════════════════\033[0m")
	fmt.Println(res.FinalOutput)
	fmt.Println()
}

func pickDebateModel(input, fallback string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return fallback
	}
	for i, m := range popularDebateModels {
		if input == fmt.Sprintf("%d", i+1) || strings.EqualFold(input, m.id) || strings.EqualFold(input, m.name) {
			return m.id
		}
	}
	return input
}
