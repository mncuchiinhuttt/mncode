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

	task := strings.TrimSpace(args)
	if task == "" {
		fmt.Print("\n\033[1;36m⚔️  Enter problem / architecture topic to debate:\033[0m\n> ")
		task = strings.TrimSpace(readLineRaw())
		if task == "" {
			fmt.Println("\033[33m[Cancelled] No debate topic provided.\033[0m")
			return
		}
	}

	fmt.Println("\n\033[1;36m═══ AI Agent Debate Arena Setup ═══\033[0m")
	fmt.Println("\033[2mSelect debater models to argue and stress-test the solution.\033[0m")
	fmt.Println()
	// 1. Proposer Model Selection
	fmt.Println("\033[1;33m1. Select Proposer Model (Author of Solution):\033[0m")
	for i, m := range popularDebateModels {
		fmt.Printf("   %d) %-34s \033[2m(%s)\033[0m\n", i+1, m.name, m.id)
	}
	fmt.Print("Choose Proposer [1-8, default 1: Claude Sonnet 4.6]: ")
	propChoice := strings.TrimSpace(readLineRaw())
	modelProposer := pickDebateModel(propChoice, "claude-sonnet-4-6")

	// 2. Critic Model Selection
	fmt.Println("\n\033[1;31m2. Select Critic Model (Devil's Advocate / Stress-Tester):\033[0m")
	for i, m := range popularDebateModels {
		fmt.Printf("   %d) %-34s \033[2m(%s)\033[0m\n", i+1, m.name, m.id)
	}
	fmt.Print("Choose Critic [1-8, default 2: DeepSeek R1]: ")
	critChoice := strings.TrimSpace(readLineRaw())
	modelCritic := pickDebateModel(critChoice, "deepseek-reasoner")

	// 3. Judge / Decider Selection
	fmt.Println("\n\033[1;36m3. Select Judge / Decider Model (Synthesizes Final Consensus):\033[0m")
	for i, m := range popularDebateModels {
		fmt.Printf("   %d) %-34s \033[2m(%s)\033[0m\n", i+1, m.name, m.id)
	}
	fmt.Print("Choose Judge [1-8, default 5: Claude Opus 4.6]: ")
	judgeChoice := strings.TrimSpace(readLineRaw())
	modelJudge := pickDebateModel(judgeChoice, "claude-opus-4-6-thinking")

	// 4. Create Dynamic Debate Combo
	now := time.Now()
	debateCombo := combos.Combo{
		ID:              fmt.Sprintf("arena-debate-%d", now.UnixNano()),
		Name:            fmt.Sprintf("Debate Arena: %s vs %s", modelProposer, modelCritic),
		Description:     fmt.Sprintf("Adversarial debate on: %s", task),
		Mode:            combos.ModeDebate,
		MaxDebateRounds: 2,
		Members: []combos.ComboMember{
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
		},
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
