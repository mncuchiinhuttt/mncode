package combos

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MemberExecutor defines the execution boundary for a single combo role turn.
type MemberExecutor interface {
	ExecuteMember(ctx context.Context, member ComboMember, model string, prompt string) (string, error)
}

// ComboListener receives lifecycle and step progress events during combo execution.
type ComboListener interface {
	FallbackLogger
	OnComboStart(comboID string, name string, mode ExecutionMode, memberCount int)
	OnComboStepStart(comboID string, role string, model string, stepIndex int, totalSteps int)
	OnComboStepDone(comboID string, role string, modelUsed string, duration time.Duration, output string)
	OnComboStepError(comboID string, role string, err error)
	OnComboComplete(comboID string, duration time.Duration, output string)
}

// ComboStepResult records the output and performance metrics of a single role step.
type ComboStepResult struct {
	StepIndex int           `json:"stepIndex"`
	Role      string        `json:"role"`
	ModelUsed string        `json:"modelUsed"`
	Duration  time.Duration `json:"duration"`
	Output    string        `json:"output"`
	Error     error         `json:"error,omitempty"`
}

// ComboResult holds the overall execution trace and final output of a Combo run.
type ComboResult struct {
	ComboID     string            `json:"comboId"`
	Mode        ExecutionMode     `json:"mode"`
	StartTime   time.Time         `json:"startTime"`
	EndTime     time.Time         `json:"endTime"`
	Steps       []ComboStepResult `json:"steps"`
	FinalOutput string            `json:"finalOutput"`
	Error       error             `json:"error,omitempty"`
}

// Runner coordinates multi-agent combo execution.
type Runner struct {
	store    *Store
	executor MemberExecutor
	listener ComboListener
}

// NewRunner creates a new Combo Runner instance.
func NewRunner(store *Store, executor MemberExecutor, listener ComboListener) *Runner {
	return &Runner{
		store:    store,
		executor: executor,
		listener: listener,
	}
}

// Run executes a combo identified by ID against the provided user prompt.
func (r *Runner) Run(ctx context.Context, comboID string, userPrompt string) (*ComboResult, error) {
	if r.store == nil {
		return nil, fmt.Errorf("combo store is required")
	}
	if r.executor == nil {
		return nil, fmt.Errorf("member executor is required")
	}
	userPrompt = strings.TrimSpace(userPrompt)
	if userPrompt == "" {
		return nil, fmt.Errorf("user prompt cannot be empty")
	}

	combo, ok := r.store.Get(comboID)
	if !ok {
		return nil, fmt.Errorf("combo %q not found", comboID)
	}

	start := time.Now()
	if r.listener != nil {
		r.listener.OnComboStart(combo.ID, combo.Name, combo.Mode, len(combo.Members))
	}

	var result *ComboResult
	var runErr error

	switch combo.Mode {
	case ModePipeline:
		result, runErr = r.runPipeline(ctx, combo, userPrompt)
	case ModeDebate:
		result, runErr = r.runDebate(ctx, combo, userPrompt)
	case ModeFanOut:
		result, runErr = r.runFanOut(ctx, combo, userPrompt)
	default:
		result, runErr = r.runPipeline(ctx, combo, userPrompt)
	}

	totalDur := time.Since(start)
	if runErr == nil && r.listener != nil && result != nil {
		r.listener.OnComboComplete(combo.ID, totalDur, result.FinalOutput)
	}

	return result, runErr
}
