package persistence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAcquireLeaseDoesNotStealActiveLease(t *testing.T) {
	store := testStore(t)
	now := time.Now().UTC()
	first := LeaseRecord{ID: "job-1", RunID: "run-a", Holder: "holder-a", AcquiredAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := store.AcquireLease(context.Background(), first); err != nil {
		t.Fatalf("first lease: %v", err)
	}
	second := first
	second.RunID = "run-b"
	second.Holder = "holder-b"
	if err := store.AcquireLease(context.Background(), second); !errors.Is(err, ErrLeaseHeld) {
		t.Fatalf("second lease error = %v, want ErrLeaseHeld", err)
	}
	if err := store.ReleaseLease(context.Background(), first.ID, first.Holder); err != nil {
		t.Fatalf("release first lease: %v", err)
	}
	if err := store.AcquireLease(context.Background(), second); err != nil {
		t.Fatalf("expired/released lease acquisition: %v", err)
	}
}
