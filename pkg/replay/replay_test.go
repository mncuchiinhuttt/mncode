package replay

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"mncode/pkg/provider"
)

func TestRecorderPersistsOrderedEventsAndForks(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := store.Start(context.Background(), "session-test", Trace{Model: "test"})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = recorder.Record(KindMessage, 1, provider.Message{Role: provider.RoleUser, Content: fmt.Sprintf("prompt %d", i)})
		}(i)
	}
	wg.Wait()
	if err := recorder.Close(true); err != nil {
		t.Fatal(err)
	}
	list, err := store.List(context.Background())
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	trace, events, err := store.Load(context.Background(), list[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 12 || !trace.Complete {
		t.Fatalf("unexpected trace: %+v events=%d", trace, len(events))
	}
	for i, event := range events {
		if event.Seq != int64(i+1) {
			t.Fatalf("event sequence %d at %d", event.Seq, i)
		}
	}
	fork, err := store.Fork(context.Background(), ForkRequest{TraceID: trace.ID, At: int64(len(events))})
	if err != nil {
		t.Fatal(err)
	}
	if len(fork.History) != 10 || fork.SessionID == trace.SessionID {
		t.Fatalf("unexpected fork: %+v", fork)
	}
}

func TestRecorderRedactsImagesAndSecrets(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := store.Start(context.Background(), "session-test", Trace{})
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Record(KindPrompt, 1, map[string]any{"authorization": "Bearer sk-proj-12345678901234567890123456789012", "image": "raw"}); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Close(true); err != nil {
		t.Fatal(err)
	}
	_, events, err := store.Load(context.Background(), "trace-1-2")
	if err == nil || events != nil {
		t.Fatal("expected generated id lookup to fail")
	}
	list, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, events, err = store.Load(context.Background(), list[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(events[0].Data) == "" || string(events[0].Data) == "raw" {
		t.Fatalf("unexpected redacted payload: %s", events[0].Data)
	}
}
func TestRecorderRedactsCamelCaseThoughtSignatureAndListsIncomplete(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := store.Start(context.Background(), "session-test", Trace{})
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Record(KindToolCall, 1, map[string]any{"thoughtSignature": "private-signature", "content": "safe"}); err != nil {
		t.Fatal(err)
	}
	traces, err := store.List(context.Background())
	if err != nil || len(traces) != 1 || traces[0].Complete {
		t.Fatalf("expected incomplete trace: %+v %v", traces, err)
	}
	_, events, err := store.Load(context.Background(), traces[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if strings.Contains(string(event.Data), "private-signature") {
			t.Fatal("thought signature leaked")
		}
	}
	_ = recorder.Close(false)
}

func TestForkRejectsForeignRoleAndParentID(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := store.Start(context.Background(), "session-test", Trace{})
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Record(KindMessage, 1, provider.Message{Role: provider.Role("hacker"), Content: "bad"}); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Close(true); err != nil {
		t.Fatal(err)
	}
	traces, err := store.List(context.Background())
	if err != nil || len(traces) != 1 {
		t.Fatalf("list: %v", err)
	}
	trace, _, err := store.Load(context.Background(), traces[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Fork(context.Background(), ForkRequest{TraceID: trace.ID, NewSessionID: trace.SessionID}); err == nil {
		t.Fatal("expected parent id rejection")
	}
	if _, err := store.Fork(context.Background(), ForkRequest{TraceID: trace.ID}); err == nil {
		t.Fatal("expected malformed role rejection")
	}
}
