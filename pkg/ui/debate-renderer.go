package ui

import (
	"fmt"
	"strings"
	"time"

	"mncode/pkg/combos"
)

type debateArenaHUD struct {
	modelA string
	modelB string
	modelC string
}

func newDebateArenaHUD(modelA, modelB, modelC string) combos.ComboListener {
	return &debateArenaHUD{
		modelA: modelA,
		modelB: modelB,
		modelC: modelC,
	}
}

func (h *debateArenaHUD) OnModelFallback(role, fromModel, toModel string, cause error) {
	fmt.Printf("\n  \033[1;33m[WARN: Fallback] %s (%s) -> Switching to: %s (%v)\033[0m\n\n", role, fromModel, toModel, cause)
}

func (h *debateArenaHUD) OnComboStart(comboID, name string, mode combos.ExecutionMode, memberCount int) {
	fmt.Println("\n\033[1;35m+---------------------------------------------------------------------------+\033[0m")
	fmt.Println("\033[1;35m|\033[0m                     \033[1;37m[AI AGENT DEBATE ARENA]\033[0m                               \033[1;35m|\033[0m")
	fmt.Printf("\033[1;35m|\033[0m   \033[1;33mProposer:\033[0m %-20s \033[1;31mCritic:\033[0m %-20s  \033[1;35m|\033[0m\n", truncateStr(h.modelA, 20), truncateStr(h.modelB, 20))
	if h.modelC != "" && h.modelC != h.modelA {
		fmt.Printf("\033[1;35m|\033[0m   \033[1;36mJudge:\033[0m %-54s \033[1;35m|\033[0m\n", truncateStr(h.modelC, 54))
	}
	fmt.Println("\033[1;35m+---------------------------------------------------------------------------+\033[0m")
}

func (h *debateArenaHUD) OnComboStepStart(comboID, role, model string, stepIndex, totalSteps int) {
	color := "\033[1;33m"
	if strings.Contains(strings.ToLower(role), "critique") || strings.Contains(strings.ToLower(role), "critic") {
		color = "\033[1;31m"
	} else if strings.Contains(strings.ToLower(role), "decision") || strings.Contains(strings.ToLower(role), "decider") {
		color = "\033[1;36m"
	}
	fmt.Printf("%s> [%s]\033[0m \033[2m(model: %s)\033[0m\n", color, role, model)
}

func (h *debateArenaHUD) OnComboStepDone(comboID, role, modelUsed string, duration time.Duration, output string) {
	durStr := fmt.Sprintf("%.1fs", duration.Seconds())
	fmt.Printf("\033[32m[OK] Completed %s (%s)\033[0m\n", role, durStr)
	fmt.Println(strings.Repeat("-", 75))
	fmt.Println(strings.TrimSpace(output))
	fmt.Println(strings.Repeat("-", 75))
	fmt.Println()
}

func (h *debateArenaHUD) OnComboStepError(comboID, role string, err error) {
	fmt.Printf("\033[1;31m[ERROR: %s] Failed: %v\033[0m\n\n", role, err)
}

func (h *debateArenaHUD) OnComboComplete(comboID string, duration time.Duration, output string) {
	durStr := fmt.Sprintf("%.1fs", duration.Seconds())
	fmt.Printf("\033[1;32m[DEBATE CONCLUDED in %s] --------------------------------------------\033[0m\n\n", durStr)
}

func truncateStr(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
