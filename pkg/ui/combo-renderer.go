package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mncode/pkg/agent"
	"mncode/pkg/combos"
	"mncode/pkg/skills"
)

type sessionComboExecutor struct {
	session *agent.Session
}

func newSessionComboExecutor(session *agent.Session) combos.MemberExecutor {
	return &sessionComboExecutor{session: session}
}

func (e *sessionComboExecutor) ExecuteMember(ctx context.Context, member combos.ComboMember, model string, prompt string) (string, error) {
	if e.session == nil {
		return "", fmt.Errorf("session is required")
	}

	baseAgent := member.BaseAgent
	if baseAgent == "" {
		baseAgent = "coder"
	}

	// Register or override agent prompt in catalog if overlay is provided
	if member.PromptOverlay != "" && e.session.Catalog != nil {
		customAgent := &skills.Agent{
			Name:        member.Role,
			Role:        member.Role,
			Description: fmt.Sprintf("Combo subagent role %s", member.Role),
			Prompt:      member.PromptOverlay,
		}
		e.session.Catalog.Agents[member.Role] = customAgent
		baseAgent = member.Role
	}

	runner := &agent.SubagentRunner{ParentSession: e.session}
	return runner.Run(ctx, baseAgent, prompt)
}

type terminalComboHUD struct{}

func newTerminalComboHUD() combos.ComboListener {
	return &terminalComboHUD{}
}

func (h *terminalComboHUD) OnModelFallback(role, fromModel, toModel string, cause error) {
	fmt.Printf("\n  \033[1;33m[WARN]  [Combo Fallback] Role '%s' model '%s' encountered an issue (%v)\033[0m\n", role, fromModel, cause)
	fmt.Printf("     \033[1;36m-> Seamlessly switching to fallback model: '%s'...\033[0m\n\n", toModel)
}

func (h *terminalComboHUD) OnComboStart(comboID, name string, mode combos.ExecutionMode, memberCount int) {
	fmt.Printf("\033[1;36m┌── [Combo: %s] (%s mode, %d agents) ──────────────────────\033[0m\n", name, mode, memberCount)
}

func (h *terminalComboHUD) OnComboStepStart(comboID, role, model string, stepIndex, totalSteps int) {
	stepStr := ""
	if totalSteps > 0 {
		stepStr = fmt.Sprintf("Step %d/%d", stepIndex, totalSteps)
	} else {
		stepStr = fmt.Sprintf("Turn %d", stepIndex)
	}
	fmt.Printf("\033[2m│\033[0m \033[1;33m● %-10s\033[0m \033[1;37m%-16s\033[0m \033[2m(model: %s)\033[0m — Running...\n", stepStr, role, model)
}

func (h *terminalComboHUD) OnComboStepDone(comboID, role, modelUsed string, duration time.Duration, output string) {
	durStr := fmt.Sprintf("%.1fs", duration.Seconds())
	summary := strings.Split(strings.TrimSpace(output), "\n")[0]
	if len(summary) > 48 {
		summary = summary[:45] + "..."
	}
	fmt.Printf("\033[2m│\033[0m \033[1;32m[OK] %-16s\033[0m \033[2m(%s | %s) — %s\033[0m\n", role, modelUsed, durStr, summary)
}

func (h *terminalComboHUD) OnComboStepError(comboID, role string, err error) {
	fmt.Printf("\033[2m│\033[0m \033[1;31m✖ %-16s — Error: %v\033[0m\n", role, err)
}

func (h *terminalComboHUD) OnComboComplete(comboID string, duration time.Duration, output string) {
	durStr := fmt.Sprintf("%.1fs", duration.Seconds())
	fmt.Printf("\033[1;36m└── [Finished in %s] ────────────────────────────────────────\033[0m\n\n", durStr)
}
