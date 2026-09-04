package agent

import (
	"errors"
	"testing"
)

type failingEventRecorder struct{ calls int }

func (r *failingEventRecorder) RecordAgentEvent(string, int, any) error {
	r.calls++
	return errors.New("disk full")
}

func TestRecordEventStopsAfterRecorderFailure(t *testing.T) {
	recorder := &failingEventRecorder{}
	session := &Session{Recorder: recorder}
	session.recordEvent("token", 1, "x")
	session.recordEvent("token", 1, "y")
	if recorder.calls != 1 || !session.recorderStop {
		t.Fatalf("recorder failure was not latched: calls=%d stopped=%t", recorder.calls, session.recorderStop)
	}
}
