package combos

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// runFanOut executes worker members concurrently and merges their outputs.
func (r *Runner) runFanOut(ctx context.Context, combo *Combo, userPrompt string) (*ComboResult, error) {
	result := &ComboResult{
		ComboID:   combo.ID,
		Mode:      combo.Mode,
		StartTime: time.Now(),
		Steps:     make([]ComboStepResult, 0),
	}

	members := combo.Members
	if len(members) == 0 {
		return result, fmt.Errorf("combo %q has no members", combo.ID)
	}

	// If there's more than 1 member, last member acts as integrator
	var workers []ComboMember
	var integrator *ComboMember

	if len(members) > 1 {
		workers = members[:len(members)-1]
		last := members[len(members)-1]
		integrator = &last
	} else {
		workers = members
	}

	type workerOutcome struct {
		index     int
		role      string
		modelUsed string
		duration  time.Duration
		output    string
		err       error
	}

	outcomes := make([]workerOutcome, len(workers))
	var wg sync.WaitGroup

	for i, m := range workers {
		wg.Add(1)
		go func(idx int, member ComboMember) {
			defer wg.Done()
			primaryModel, fallbackModel := ResolveRoleModels(member)
			stepNum := idx + 1

			if r.listener != nil {
				r.listener.OnComboStepStart(combo.ID, member.Role, primaryModel, stepNum, len(members))
			}

			start := time.Now()
			workerPrompt := fmt.Sprintf(
				"[Parallel Worker: %s]\nTask:\n%s",
				member.Role, userPrompt,
			)
			if member.PromptOverlay != "" {
				workerPrompt += fmt.Sprintf("\n\n[Role Instructions]:\n%s", member.PromptOverlay)
			}

			out, usedModel, err := ExecuteWithModelFallback(
				ctx, member.Role, primaryModel, fallbackModel, r.listener,
				func(stepCtx context.Context, effectiveModel string) (string, error) {
					return r.executor.ExecuteMember(stepCtx, member, effectiveModel, workerPrompt)
				},
			)

			dur := time.Since(start)
			outcomes[idx] = workerOutcome{
				index: stepNum, role: member.Role, modelUsed: usedModel,
				duration: dur, output: out, err: err,
			}

			if err == nil && r.listener != nil {
				r.listener.OnComboStepDone(combo.ID, member.Role, usedModel, dur, out)
			}
		}(i, m)
	}

	wg.Wait()

	var parallelArtifacts strings.Builder
	for _, o := range outcomes {
		result.Steps = append(result.Steps, ComboStepResult{
			StepIndex: o.index, Role: o.role, ModelUsed: o.modelUsed,
			Duration: o.duration, Output: o.output, Error: o.err,
		})
		if o.err != nil {
			result.EndTime = time.Now()
			result.Error = o.err
			return result, fmt.Errorf("parallel worker %s failed: %w", o.role, o.err)
		}
		parallelArtifacts.WriteString(fmt.Sprintf("\n--- [Worker Output: %s (%s)] ---\n%s\n", o.role, o.modelUsed, o.output))
	}

	// Run Integrator / Synthesizer step if configured
	if integrator != nil {
		intIdx := len(members)
		primaryModel, fallbackModel := ResolveRoleModels(*integrator)
		if r.listener != nil {
			r.listener.OnComboStepStart(combo.ID, integrator.Role, primaryModel, intIdx, len(members))
		}
		intStart := time.Now()
		mergePrompt := fmt.Sprintf(
			"[Parallel Synthesis & Merge (%s)]\nOriginal Task:\n%s\n\n[Parallel Worker Outputs]:\n%s\n\nMerge the findings and synthesize the deliverable.",
			integrator.Role, userPrompt, parallelArtifacts.String(),
		)
		mergeOut, usedModel, mergeErr := ExecuteWithModelFallback(
			ctx, integrator.Role, primaryModel, fallbackModel, r.listener,
			func(stepCtx context.Context, effectiveModel string) (string, error) {
				return r.executor.ExecuteMember(stepCtx, *integrator, effectiveModel, mergePrompt)
			},
		)
		intDur := time.Since(intStart)
		result.Steps = append(result.Steps, ComboStepResult{
			StepIndex: intIdx, Role: integrator.Role, ModelUsed: usedModel,
			Duration: intDur, Output: mergeOut, Error: mergeErr,
		})
		if mergeErr != nil {
			result.EndTime = time.Now()
			result.Error = mergeErr
			return result, fmt.Errorf("integrator step (%s) failed: %w", integrator.Role, mergeErr)
		}
		if r.listener != nil {
			r.listener.OnComboStepDone(combo.ID, integrator.Role, usedModel, intDur, mergeOut)
		}
		result.FinalOutput = mergeOut
	} else {
		result.FinalOutput = parallelArtifacts.String()
	}

	result.EndTime = time.Now()
	return result, nil
}
