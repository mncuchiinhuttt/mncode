package budget

import (
	"fmt"
	"sync"
)

// Tracker manages active session budget and enforces soft/hard limits.
type Tracker struct {
	mu             sync.RWMutex
	Spec           BudgetSpec
	InputTokens    int64
	OutputTokens   int64
	ThinkingTokens int64
	SpentTokens    int64
	warned80       bool
	warned100      bool
}

// NewTracker creates an active session budget tracker.
func NewTracker(spec BudgetSpec) *Tracker {
	return &Tracker{Spec: spec}
}

// SetBudget updates the active budget limit.
func (t *Tracker) SetBudget(spec BudgetSpec) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Spec = spec
	t.warned80 = false
	t.warned100 = false
}

// AddTokens accumulates token usage and checks for budget alerts or hard stops.
func (t *Tracker) AddTokens(input, output, thinking int) (hardExceeded bool, notice string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.InputTokens += int64(input)
	t.OutputTokens += int64(output)
	t.ThinkingTokens += int64(thinking)
	t.SpentTokens += int64(input + output + thinking)

	if t.Spec.TokenLimit <= 0 {
		return false, ""
	}

	ratio := float64(t.SpentTokens) / float64(t.Spec.TokenLimit)

	if ratio >= 1.0 {
		if t.Spec.IsHardStop {
			return true, fmt.Sprintf("[HARD STOP] Token budget exhausted (%d/%d tokens). Aborting turn.", t.SpentTokens, t.Spec.TokenLimit)
		}
		if !t.warned100 {
			t.warned100 = true
			return false, fmt.Sprintf("[WARN: Budget Exceeded] Session reached %d/%d tokens (100%%).", t.SpentTokens, t.Spec.TokenLimit)
		}
	} else if ratio >= 0.8 && !t.warned80 {
		t.warned80 = true
		return false, fmt.Sprintf("[WARN: Budget Advisory] Session token usage reached 80%% (%d/%d tokens).", t.SpentTokens, t.Spec.TokenLimit)
	}

	return false, ""
}

// IsHardStopExceeded reports whether execution must be aborted.
func (t *Tracker) IsHardStopExceeded() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.Spec.TokenLimit <= 0 || !t.Spec.IsHardStop {
		return false
	}
	return t.SpentTokens >= t.Spec.TokenLimit
}

// Remaining returns remaining token allowance and whether unlimited.
func (t *Tracker) Remaining() (int64, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.Spec.TokenLimit <= 0 {
		return 0, true
	}
	rem := t.Spec.TokenLimit - t.SpentTokens
	if rem < 0 {
		rem = 0
	}
	return rem, false
}
