package combos

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// runPipeline executes combo members sequentially in a linear pipeline.
func (r *Runner) runPipeline(ctx context.Context, combo *Combo, userPrompt string) (*ComboResult, error) {
	result := &ComboResult{
		ComboID:   combo.ID,
		Mode:      combo.Mode,
		StartTime: time.Now(),
		Steps:     make([]ComboStepResult, 0, len(combo.Members)),
	}

	var stageArtifacts strings.Builder
	lastOutput := ""

	for index, member := range combo.Members {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		stepNum := index + 1
		totalSteps := len(combo.Members)
		primaryModel, fallbackModel := ResolveRoleModels(member)

		// 1. Prepare stage prompt with aggregated context
		stepPrompt := fmt.Sprintf(
			"[Combo Pipeline: %s | Step %d/%d: %s]\nTask:\n%s",
			combo.Name, stepNum, totalSteps, member.Role, userPrompt,
		)
		if member.PromptOverlay != "" {
			stepPrompt += fmt.Sprintf("\n\n[Role Instructions - %s]:\n%s", member.Role, member.PromptOverlay)
		}
		if stageArtifacts.Len() > 0 {
			stepPrompt += fmt.Sprintf("\n\n[Context From Previous Pipeline Steps]:\n%s", stageArtifacts.String())
		}

		// 2. Notify step start
		if r.listener != nil {
			r.listener.OnComboStepStart(combo.ID, member.Role, primaryModel, stepNum, totalSteps)
		}

		stepStart := time.Now()

		// 3. Execute member with automatic model fallback
		output, usedModel, execErr := ExecuteWithModelFallback(
			ctx, member.Role, primaryModel, fallbackModel, r.listener,
			func(stepCtx context.Context, effectiveModel string) (string, error) {
				return r.executor.ExecuteMember(stepCtx, member, effectiveModel, stepPrompt)
			},
		)

		stepDuration := time.Since(stepStart)
		stepResult := ComboStepResult{
			StepIndex: stepNum,
			Role:      member.Role,
			ModelUsed: usedModel,
			Duration:  stepDuration,
			Output:    output,
			Error:     execErr,
		}
		result.Steps = append(result.Steps, stepResult)

		if execErr != nil {
			result.EndTime = time.Now()
			result.Error = execErr
			if r.listener != nil {
				r.listener.OnComboStepError(combo.ID, member.Role, execErr)
			}
			return result, fmt.Errorf("pipeline step %d (%s) failed: %w", stepNum, member.Role, execErr)
		}

		if r.listener != nil {
			r.listener.OnComboStepDone(combo.ID, member.Role, usedModel, stepDuration, output)
		}

		lastOutput = output
		stageArtifacts.WriteString(fmt.Sprintf("\n--- [Output from Step %d: %s (%s)] ---\n%s\n", stepNum, member.Role, usedModel, output))
	}

	result.EndTime = time.Now()
	result.FinalOutput = lastOutput
	return result, nil
}
