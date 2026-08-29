package orchestration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"mncode/pkg/persistence"
)

func testPersistenceStore(t *testing.T) *persistence.Store {
	t.Helper()
	s, err := persistence.Open(context.Background(), persistence.StoreConfig{
		Path: filepath.Join(t.TempDir(), "orchestration_test.db"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestLifecycleTransitions(t *testing.T) {
	if !CanTransition(StateQueued, StateRunning) {
		t.Error("expected Queued -> Running to be valid")
	}
	if !CanTransition(StateRunning, StateCompleted) {
		t.Error("expected Running -> Completed to be valid")
	}
	if CanTransition(StateCompleted, StateRunning) {
		t.Error("expected Completed -> Running to be invalid")
	}
	if CanTransition(StateFailed, StateRunning) {
		t.Error("expected Failed -> Running to be invalid")
	}

	run, err := NewRun(context.Background(), RunMeta{ID: "r1", Kind: KindForegroundTurn}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if run.State() != StateQueued {
		t.Fatalf("expected initial state Queued, got %s", run.State())
	}

	if err := run.Transition(StateRunning); err != nil {
		t.Fatalf("transition to running: %v", err)
	}

	if err := run.Transition(StateWaiting); err != nil {
		t.Fatalf("transition to waiting: %v", err)
	}

	if err := run.Transition(StateRunning); err != nil {
		t.Fatalf("transition back to running: %v", err)
	}

	if err := run.Complete("success", 10, 20); err != nil {
		t.Fatalf("complete run: %v", err)
	}

	snap := run.Snapshot()
	if snap.State != StateCompleted || snap.Result != "success" || snap.TokensIn != 10 || snap.TokensOut != 20 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestRunManagerLifecycleAndEvents(t *testing.T) {
	store := testPersistenceStore(t)
	mgr := NewRunManager(store)

	var received []EventEnvelope
	mgr.Subscribe(func(e EventEnvelope) {
		received = append(received, e)
	})

	run, err := mgr.CreateRun(context.Background(), RunMeta{
		ID:         "test-run-1",
		ChatID:     "chat-123",
		Generation: 1,
		Kind:       KindSubagent,
	})
	if err != nil {
		t.Fatal(err)
	}

	run.EmitEvent("subagent_progress", map[string]string{"status": "analyzing"})
	run.Log("Started analysis of workspace")

	if err := run.Complete("Done analysis", 100, 200); err != nil {
		t.Fatal(err)
	}

	if len(received) != 1 || received[0].Type != "subagent_progress" {
		t.Fatalf("expected 1 event received, got %d", len(received))
	}

	logs := run.Logs(10)
	if len(logs) != 1 || logs[0] != "Started analysis of workspace" {
		t.Fatalf("unexpected logs: %+v", logs)
	}

	snap, err := mgr.Wait(context.Background(), "test-run-1", 1*time.Second)
	if err != nil {
		t.Fatalf("wait run: %v", err)
	}
	if snap.State != StateCompleted {
		t.Fatalf("expected Completed, got %s", snap.State)
	}
}

func TestRunManagerCancellation(t *testing.T) {
	mgr := NewRunManager(nil)
	_, err := mgr.CreateRun(context.Background(), RunMeta{
		ID:   "cancel-run",
		Kind: KindProcess,
	})
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = mgr.Cancel("cancel-run")
	}()

	snap, err := mgr.Wait(context.Background(), "cancel-run", 500*time.Millisecond)
	if err != nil {
		t.Fatalf("wait error: %v", err)
	}
	if snap.State != StateCancelled {
		t.Fatalf("expected StateCancelled, got %s", snap.State)
	}
}

func TestScheduleLeaseExclusivity(t *testing.T) {
	store := testPersistenceStore(t)
	mgr := NewRunManager(store)
	ctx := context.Background()

	acquired, err := mgr.AcquireScheduleLease(ctx, "cron-hourly", "worker-1", 1*time.Hour)
	if err != nil || !acquired {
		t.Fatalf("worker-1 lease acquire failed: %v", err)
	}

	acquired2, err := mgr.AcquireScheduleLease(ctx, "cron-hourly", "worker-2", 1*time.Hour)
	if err != nil {
		t.Fatalf("worker-2 unexpected error: %v", err)
	}
	if acquired2 {
		t.Fatal("worker-2 should not acquire active lease held by worker-1")
	}

	if err := mgr.ReleaseScheduleLease(ctx, "cron-hourly", "worker-1"); err != nil {
		t.Fatalf("release lease: %v", err)
	}

	acquired3, err := mgr.AcquireScheduleLease(ctx, "cron-hourly", "worker-2", 1*time.Hour)
	if err != nil || !acquired3 {
		t.Fatalf("worker-2 should acquire released lease: %v", err)
	}
}
