package orchestration

import (
	"context"
	"errors"
	"fmt"
	"mncode/pkg/persistence"
	"sync"
	"time"
)

// EventSubscriber receives event envelopes.
type EventSubscriber func(EventEnvelope)

// RunManager coordinates all active runs, processes, subagents, and leases.
type RunManager struct {
	mu          sync.RWMutex
	runs        map[string]*Run
	store       persistence.RunStore
	subscribers []EventSubscriber
}

// NewRunManager creates a RunManager backed optionally by a persistence.RunStore.
func NewRunManager(store persistence.RunStore) *RunManager {
	return &RunManager{
		runs:        make(map[string]*Run),
		store:       store,
		subscribers: make([]EventSubscriber, 0, 8),
	}
}

// Subscribe registers a global event listener.
func (m *RunManager) Subscribe(sub EventSubscriber) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers = append(m.subscribers, sub)
}

// CreateRun initializes and registers a new supervised run.
func (m *RunManager) CreateRun(parentCtx context.Context, meta RunMeta) (*Run, error) {
	if meta.ID == "" {
		meta.ID = fmt.Sprintf("run-%d-%s", time.Now().UnixNano(), meta.Kind)
	}

	m.mu.Lock()
	if _, exists := m.runs[meta.ID]; exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("run %q already exists", meta.ID)
	}

	run, err := NewRun(parentCtx, meta, defaultLogCapacity)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}

	run.onEvent = func(env EventEnvelope) {
		m.dispatch(env)
	}

	m.runs[meta.ID] = run
	m.mu.Unlock()

	// Persist initial record if store is available
	if m.store != nil {
		_ = m.store.SaveRun(parentCtx, run.Snapshot().ToPersistence())
	}

	return run, nil
}

// GetRun returns an active run by ID.
func (m *RunManager) GetRun(id string) (*Run, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runs[id]
	return r, ok
}

// ListActive returns snapshots of all active runs.
func (m *RunManager) ListActive() []RunSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]RunSnapshot, 0, len(m.runs))
	for _, r := range m.runs {
		out = append(out, r.Snapshot())
	}
	return out
}

// Cancel terminates a run by ID.
func (m *RunManager) Cancel(id string) error {
	m.mu.RLock()
	r, ok := m.runs[id]
	m.mu.RUnlock()
	if !ok {
		return ErrRunNotFound
	}
	r.Cancel()
	if m.store != nil {
		_ = m.store.SaveRun(context.Background(), r.Snapshot().ToPersistence())
	}
	return nil
}

// Wait blocks until the run completes, fails, is cancelled, or the timeout expires.
func (m *RunManager) Wait(ctx context.Context, id string, timeout time.Duration) (RunSnapshot, error) {
	m.mu.RLock()
	r, ok := m.runs[id]
	m.mu.RUnlock()
	if !ok {
		return RunSnapshot{}, ErrRunNotFound
	}

	var timerCh <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		timerCh = timer.C
	}

	select {
	case <-r.Done():
		snap := r.Snapshot()
		if m.store != nil {
			_ = m.store.SaveRun(context.Background(), snap.ToPersistence())
		}
		return snap, nil
	case <-timerCh:
		return r.Snapshot(), ErrRunTimeout
	case <-ctx.Done():
		return r.Snapshot(), ctx.Err()
	}
}

// AcquireScheduleLease attempts to claim a cross-process schedule execution lock.
func (m *RunManager) AcquireScheduleLease(ctx context.Context, jobID, holder string, duration time.Duration) (bool, error) {
	if m.store == nil {
		return true, nil // Memory-only fallback
	}
	now := time.Now().UTC()
	lease := persistence.LeaseRecord{
		ID:         "schedule:" + jobID,
		RunID:      jobID,
		Holder:     holder,
		AcquiredAt: now,
		ExpiresAt:  now.Add(duration),
	}
	err := m.store.AcquireLease(ctx, lease)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, persistence.ErrLeaseHeld) {
		return false, nil
	}
	return false, err
}

// ReleaseScheduleLease releases a schedule lock.
func (m *RunManager) ReleaseScheduleLease(ctx context.Context, jobID, holder string) error {
	if m.store == nil {
		return nil
	}
	return m.store.ReleaseLease(ctx, "schedule:"+jobID, holder)
}

func (m *RunManager) dispatch(env EventEnvelope) {
	m.mu.RLock()
	subs := append([]EventSubscriber(nil), m.subscribers...)
	m.mu.RUnlock()

	for _, sub := range subs {
		sub(env)
	}

	// Persist event if backing store exists
	if m.store != nil {
		_ = m.store.AppendEvent(context.Background(), persistence.EventRecord{
			RunID:     env.RunID,
			JobID:     env.JobID,
			Sequence:  env.Sequence,
			Type:      env.Type,
			Payload:   env.Payload,
			CreatedAt: env.Timestamp,
		})
	}
}
