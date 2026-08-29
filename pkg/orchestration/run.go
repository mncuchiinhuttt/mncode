package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

const defaultLogCapacity = 500

// Run represents an active supervised unit of work.
type Run struct {
	mu         sync.RWMutex
	meta       RunMeta
	state      RunState
	startedAt  time.Time
	finishedAt *time.Time
	err        error
	result     string
	exitCode   int
	tokensIn   int
	tokensOut  int

	ctx        context.Context
	cancel     context.CancelFunc
	doneCh     chan struct{}
	seqCounter int
	logs       []string
	logCap     int

	onEvent func(EventEnvelope)
}

// NewRun creates a new managed Run in StateQueued.
func NewRun(parentCtx context.Context, meta RunMeta, logCapacity int) (*Run, error) {
	if err := meta.Validate(); err != nil {
		return nil, err
	}
	if logCapacity <= 0 {
		logCapacity = defaultLogCapacity
	}
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(parentCtx)

	return &Run{
		meta:      meta,
		state:     StateQueued,
		startedAt: time.Now().UTC(),
		ctx:       ctx,
		cancel:    cancel,
		doneCh:    make(chan struct{}),
		logCap:    logCapacity,
		logs:      make([]string, 0, 64),
	}, nil
}

// ID returns the run ID.
func (r *Run) ID() string { return r.meta.ID }

// Context returns the run context.
func (r *Run) Context() context.Context { return r.ctx }

// State returns the current RunState.
func (r *Run) State() RunState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

// Snapshot returns an immutable copy of the run state.
func (r *Run) Snapshot() RunSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	errMsg := ""
	if r.err != nil {
		errMsg = r.err.Error()
	}

	return RunSnapshot{
		Meta:       r.meta,
		State:      r.state,
		StartedAt:  r.startedAt,
		FinishedAt: r.finishedAt,
		Error:      errMsg,
		Result:     r.result,
		ExitCode:   r.exitCode,
		TokensIn:   r.tokensIn,
		TokensOut:  r.tokensOut,
	}
}

// Transition moves the run into a new state if allowed.
func (r *Run) Transition(to RunState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !CanTransition(r.state, to) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrIllegalTransition, r.state, to)
	}
	r.state = to
	return nil
}

// Log records a log line in the bounded log buffer.
func (r *Run) Log(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	r.mu.Lock()
	if len(r.logs) >= r.logCap {
		// Evict oldest
		r.logs = append(r.logs[1:], line)
	} else {
		r.logs = append(r.logs, line)
	}
	r.mu.Unlock()
}

// Logs returns a copy of the captured log lines.
func (r *Run) Logs(tail int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if tail <= 0 || tail >= len(r.logs) {
		return append([]string(nil), r.logs...)
	}
	return append([]string(nil), r.logs[len(r.logs)-tail:]...)
}

// EmitEvent emits an event envelope to the supervisor and listeners.
func (r *Run) EmitEvent(eventType string, payload interface{}) EventEnvelope {
	r.mu.Lock()
	r.seqCounter++
	seq := r.seqCounter
	cb := r.onEvent
	r.mu.Unlock()

	var raw json.RawMessage
	if payload != nil {
		raw, _ = json.Marshal(payload)
	}

	env := EventEnvelope{
		RunID:      r.meta.ID,
		ChatID:     r.meta.ChatID,
		Generation: r.meta.Generation,
		Sequence:   seq,
		Type:       eventType,
		Payload:    raw,
		Timestamp:  time.Now().UTC(),
	}

	if cb != nil {
		cb(env)
	}
	return env
}

// Cancel requests cancellation of the run and its children.
func (r *Run) Cancel() {
	r.mu.Lock()
	if !r.state.IsTerminal() {
		r.state = StateCancelled
		now := time.Now().UTC()
		r.finishedAt = &now
		r.err = ErrRunCancelled
	}
	r.mu.Unlock()
	r.cancel()
	select {
	case <-r.doneCh:
	default:
		close(r.doneCh)
	}
}

// Complete marks the run successfully finished with result.
func (r *Run) Complete(result string, tokensIn, tokensOut int) error {
	r.mu.Lock()
	if r.state.IsTerminal() {
		r.mu.Unlock()
		return fmt.Errorf("%w: already terminal (%s)", ErrIllegalTransition, r.state)
	}
	r.state = StateCompleted
	now := time.Now().UTC()
	r.finishedAt = &now
	r.result = result
	r.tokensIn = tokensIn
	r.tokensOut = tokensOut
	r.mu.Unlock()

	select {
	case <-r.doneCh:
	default:
		close(r.doneCh)
	}
	return nil
}

// Fail marks the run as failed with an error.
func (r *Run) Fail(err error) error {
	if err == nil {
		err = errors.New("unspecified run failure")
	}
	r.mu.Lock()
	if r.state.IsTerminal() {
		r.mu.Unlock()
		return fmt.Errorf("%w: already terminal (%s)", ErrIllegalTransition, r.state)
	}
	r.state = StateFailed
	now := time.Now().UTC()
	r.finishedAt = &now
	r.err = err
	r.mu.Unlock()

	select {
	case <-r.doneCh:
	default:
		close(r.doneCh)
	}
	return nil
}

// Done returns a channel that closes when the run reaches a terminal state.
func (r *Run) Done() <-chan struct{} {
	return r.doneCh
}
