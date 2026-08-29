package combos

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// runDebate executes an adversarial/constructive debate between members before final synthesis.
func (r *Runner) runDebate(ctx context.Context, combo *Combo, userPrompt string) (*ComboResult, error) {
	result := &ComboResult{
		ComboID:   combo.ID,
		Mode:      combo.Mode,
		StartTime: time.Now(),
		Steps:     make([]ComboStepResult, 0),
	}

	proposer, critic, decider := findDebateRoles(combo.Members)
	rounds := combo.MaxDebateRounds
	if rounds <= 0 {
		rounds = 2
	}

	var debateTranscript strings.Builder
	debateTranscript.WriteString(fmt.Sprintf("Original User Task:\n%s\n\n", userPrompt))

	stepIndex := 0

	for round := 1; round <= rounds; round++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// 1. Proposer Turn
		stepIndex++
		proposerPrompt := fmt.Sprintf(
			"[Debate Round %d/%d - Proposal Phase (%s)]\nTask: %s\n\n[Debate History]:\n%s\n\nProvide your refined proposal/code addressing any prior critiques.",
			round, rounds, proposer.Role, userPrompt, debateTranscript.String(),
		)
		pOutput, pModel, pErr := r.runDebateTurn(ctx, combo.ID, proposer, stepIndex, proposerPrompt)
		if pErr != nil {
			return result, fmt.Errorf("debate round %d proposer failed: %w", round, pErr)
		}
		result.Steps = append(result.Steps, ComboStepResult{
			StepIndex: stepIndex, Role: fmt.Sprintf("%s (Round %d Proposal)", proposer.Role, round),
			ModelUsed: pModel, Output: pOutput,
		})
		debateTranscript.WriteString(fmt.Sprintf("\n--- [Round %d: %s Proposal (%s)] ---\n%s\n", round, proposer.Role, pModel, pOutput))

		// 2. Critic Turn
		stepIndex++
		criticPrompt := fmt.Sprintf(
			"[Debate Round %d/%d - Critique Phase (%s)]\nTask: %s\n\n[Proposal to Evaluate]:\n%s\n\nIdentify edge cases, performance risks, security vulnerabilities, or invalid assumptions.",
			round, rounds, critic.Role, userPrompt, pOutput,
		)
		cOutput, cModel, cErr := r.runDebateTurn(ctx, combo.ID, critic, stepIndex, criticPrompt)
		if cErr != nil {
			return result, fmt.Errorf("debate round %d critic failed: %w", round, cErr)
		}
		result.Steps = append(result.Steps, ComboStepResult{
			StepIndex: stepIndex, Role: fmt.Sprintf("%s (Round %d Critique)", critic.Role, round),
			ModelUsed: cModel, Output: cOutput,
		})
		debateTranscript.WriteString(fmt.Sprintf("\n--- [Round %d: %s Critique (%s)] ---\n%s\n", round, critic.Role, cModel, cOutput))
	}

	// 3. Final Decider Synthesis
	stepIndex++
	deciderPrompt := fmt.Sprintf(
		"[Debate Final Synthesis (%s)]\nOriginal Task: %s\n\n[Full Debate Transcript]:\n%s\n\nSynthesize the debate consensus, resolve disagreements, and produce the definitive deliverable.",
		decider.Role, userPrompt, debateTranscript.String(),
	)
	dOutput, dModel, dErr := r.runDebateTurn(ctx, combo.ID, decider, stepIndex, deciderPrompt)
	if dErr != nil {
		return result, fmt.Errorf("debate decider synthesis failed: %w", dErr)
	}
	result.Steps = append(result.Steps, ComboStepResult{
		StepIndex: stepIndex, Role: fmt.Sprintf("%s (Final Decision)", decider.Role),
		ModelUsed: dModel, Output: dOutput,
	})

	result.EndTime = time.Now()
	result.FinalOutput = dOutput
	return result, nil
}

func (r *Runner) runDebateTurn(ctx context.Context, comboID string, m ComboMember, stepIdx int, prompt string) (string, string, error) {
	primaryModel, fallbackModel := ResolveRoleModels(m)
	if r.listener != nil {
		r.listener.OnComboStepStart(comboID, m.Role, primaryModel, stepIdx, 0)
	}
	start := time.Now()
	output, usedModel, err := ExecuteWithModelFallback(
		ctx, m.Role, primaryModel, fallbackModel, r.listener,
		func(stepCtx context.Context, effectiveModel string) (string, error) {
			return r.executor.ExecuteMember(stepCtx, m, effectiveModel, prompt)
		},
	)
	if err == nil && r.listener != nil {
		r.listener.OnComboStepDone(comboID, m.Role, usedModel, time.Since(start), output)
	}
	return output, usedModel, err
}

func findDebateRoles(members []ComboMember) (proposer, critic, decider ComboMember) {
	if len(members) == 0 {
		dummy := ComboMember{Role: "worker", BaseAgent: "coder"}
		return dummy, dummy, dummy
	}
	if len(members) == 1 {
		return members[0], members[0], members[0]
	}
	if len(members) == 2 {
		return members[0], members[1], members[0]
	}

	proposer = members[0]
	critic = members[1]
	decider = members[len(members)-1]

	for _, m := range members {
		r := strings.ToLower(m.Role)
		if r == RoleAdvisor || strings.Contains(r, "critic") || r == RoleCodeReviewer {
			critic = m
		} else if r == RoleArchitect || r == RoleCoder {
			proposer = m
		} else if r == RolePlanner || strings.Contains(r, "decider") || strings.Contains(r, "lead") {
			decider = m
		}
	}
	return proposer, critic, decider
}
