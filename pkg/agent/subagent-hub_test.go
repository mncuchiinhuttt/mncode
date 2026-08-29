package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"mncode/pkg/config"
	"mncode/pkg/tools"
)

func TestSubagentRegistryPeerMessageRoundTrip(t *testing.T) {
	hub := NewSubagentRegistry()
	hub.Register("one", "scout", "Scout", "inspect")
	hub.Register("two", "tester", "Tester", "test")
	if err := hub.SendMessage(context.Background(), "one", "two", "I found the entry point."); err != nil {
		t.Fatal(err)
	}
	message, err := hub.ReceiveMessage(context.Background(), "two")
	if err != nil {
		t.Fatal(err)
	}
	if message.From != "one" || message.To != "two" || message.Content == "" || message.ID == "" {
		t.Fatalf("unexpected peer message: %+v", message)
	}
	if len(hub.Messages()) != 1 {
		t.Fatalf("message log length = %d, want 1", len(hub.Messages()))
	}
}

func TestSubagentPeerToolReceiveTimeout(t *testing.T) {
	hub := NewSubagentRegistry()
	hub.Register("one", "scout", "Scout", "inspect")
	tool := &tools.SubagentPeerTool{SelfID: "one", Hub: hub}
	started := time.Now()
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "receive", "timeout_ms": 10,
	})
	if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("expected bounded timeout, got %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("peer receive exceeded bounded timeout")
	}
}

func TestSubagentPeerToolSendAndReceive(t *testing.T) {
	hub := NewSubagentRegistry()
	hub.Register("one", "scout", "Scout", "inspect")
	hub.Register("two", "tester", "Tester", "test")
	sender := &tools.SubagentPeerTool{SelfID: "one", Hub: hub}
	receiver := &tools.SubagentPeerTool{SelfID: "two", Hub: hub}
	if _, err := sender.Execute(context.Background(), map[string]interface{}{
		"action": "send", "to": "two", "message": "status ready",
	}); err != nil {
		t.Fatal(err)
	}
	body, err := receiver.Execute(context.Background(), map[string]interface{}{
		"action": "receive", "timeout_ms": 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "status ready") {
		t.Fatalf("received body = %q", body)
	}
}

func TestSubagentPeerSendFailsFastForUnknownOrFullPeer(t *testing.T) {
	hub := NewSubagentRegistry()
	hub.Register("one", "scout", "Scout", "inspect")
	hub.Register("two", "tester", "Tester", "test")
	if err := hub.SendMessage(context.Background(), "one", "missing", "hello"); !errors.Is(err, ErrPeerUnavailable) {
		t.Fatalf("unknown peer error = %v", err)
	}
	for index := 0; index < 32; index++ {
		if err := hub.SendMessage(context.Background(), "one", "two", "message"); err != nil {
			t.Fatalf("fill inbox at %d: %v", index, err)
		}
	}
	if err := hub.SendMessage(context.Background(), "one", "two", "overflow"); !errors.Is(err, ErrPeerInboxFull) {
		t.Fatalf("overflow error = %v", err)
	}
}

func TestSubagentConfigCloneDoesNotAliasParent(t *testing.T) {
	parent := &config.Config{
		Settings: map[string]string{"effort": "high"},
		CustomProviders: map[string]config.CustomProvider{
			"local": {ID: "local", Models: []config.CustomModel{{ID: "one"}}},
		},
	}
	clone, err := cloneSessionConfig(parent)
	if err != nil {
		t.Fatal(err)
	}
	clone.Settings["effort"] = "low"
	clone.CustomProviders["local"] = config.CustomProvider{ID: "changed"}
	if parent.Settings["effort"] != "high" || parent.CustomProviders["local"].ID != "local" {
		t.Fatal("subagent config mutation changed parent config")
	}
}
