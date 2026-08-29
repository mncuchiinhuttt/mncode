package agent

import (
	"context"
	"io"
	"strings"
	"testing"

	"mncode/pkg/config"
	"mncode/pkg/provider"
	"mncode/pkg/tools"
)

type loopingProvider struct {
	calls int
}

func (p *loopingProvider) Name() string { return "looping-test-provider" }

func (p *loopingProvider) Stream(_ context.Context, _ *provider.CompletionRequest, _ func(provider.StreamEvent) error) (*provider.CompletionResponse, error) {
	p.calls++
	return &provider.CompletionResponse{
		ToolCalls: []provider.ToolCall{{
			ID:        "loop-tool-call",
			Name:      "loop_test_tool",
			Arguments: map[string]interface{}{},
		}},
	}, nil
}

type retryingProvider struct {
	calls     int
	failures  int
	retryable bool
}

func (p *retryingProvider) Name() string { return "retrying-test-provider" }

func (p *retryingProvider) Stream(context.Context, *provider.CompletionRequest, func(provider.StreamEvent) error) (*provider.CompletionResponse, error) {
	p.calls++
	if p.calls <= p.failures {
		return nil, &provider.ProviderError{
			Provider:   p.Name(),
			StatusCode: 503,
			Class:      provider.ErrorClassServer,
			Retryable:  p.retryable,
			Message:    "temporary failure",
		}
	}
	return &provider.CompletionResponse{Content: "ok"}, nil
}

func TestStreamProviderRetriesRetryableFailuresWithinBudget(t *testing.T) {
	p := &retryingProvider{failures: 2, retryable: true}
	session := &Session{Provider: p}

	resp, err := session.streamProvider(context.Background(), &provider.CompletionRequest{}, nil)
	if err != nil {
		t.Fatalf("streamProvider() error = %v", err)
	}
	if resp == nil || resp.Content != "ok" {
		t.Fatalf("streamProvider() response = %+v, want successful response", resp)
	}
	if p.calls != 3 {
		t.Fatalf("provider calls = %d, want three attempts", p.calls)
	}
}

func TestStreamProviderDoesNotRetryNonRetryableFailures(t *testing.T) {
	p := &retryingProvider{failures: 1, retryable: false}
	session := &Session{Provider: p}

	_, err := session.streamProvider(context.Background(), &provider.CompletionRequest{}, nil)
	// A provider error classified as non-retryable must be returned immediately
	// rather than being sent through account rotation or cooldown.
	if err == nil {
		t.Fatal("streamProvider() error = nil, want non-retryable provider failure")
	}
	if p.calls != 1 {
		t.Fatalf("provider calls = %d, want one attempt", p.calls)
	}
}

func TestStreamProviderHonorsCancellationBeforeAttempt(t *testing.T) {
	p := &retryingProvider{retryable: true}
	session := &Session{Provider: p}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := session.streamProvider(ctx, &provider.CompletionRequest{}, nil)
	if err != context.Canceled {
		t.Fatalf("streamProvider() error = %v, want context.Canceled", err)
	}
	if p.calls != 0 {
		t.Fatalf("provider calls = %d, want zero after cancellation", p.calls)
	}
}

type scrubberProvider struct{}

func (scrubberProvider) Name() string { return "scrubber-test-provider" }

func (scrubberProvider) Stream(_ context.Context, _ *provider.CompletionRequest, cb func(provider.StreamEvent) error) (*provider.CompletionResponse, error) {
	for _, text := range []string{"visible <memory-", "context>private", " value</memory-", "context> tail"} {
		if err := cb(provider.StreamEvent{Type: provider.EventToken, Text: text}); err != nil {
			return nil, err
		}
	}
	return &provider.CompletionResponse{Content: "visible <local_memories>secret</local_memories> tail"}, nil
}

func TestStreamProviderScrubsPrivateMemoryContext(t *testing.T) {
	var visible strings.Builder
	session := &Session{Provider: scrubberProvider{}}
	resp, err := session.streamProvider(context.Background(), &provider.CompletionRequest{}, func(event provider.StreamEvent) error {
		visible.WriteString(event.Text)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || strings.Contains(resp.Content, "secret") {
		t.Fatalf("response leaked private memory context: %+v", resp)
	}
	if got, want := visible.String(), "visible  tail"; got != want {
		t.Fatalf("visible stream = %q, want %q", got, want)
	}
}

type partialRetryProvider struct{ calls int }

func (p *partialRetryProvider) Name() string { return "partial-retry-provider" }

func (p *partialRetryProvider) Stream(_ context.Context, _ *provider.CompletionRequest, cb func(provider.StreamEvent) error) (*provider.CompletionResponse, error) {
	p.calls++
	if p.calls == 1 {
		_ = cb(provider.StreamEvent{Type: provider.EventToken, Text: "partial output"})
		return nil, io.ErrUnexpectedEOF
	}
	_ = cb(provider.StreamEvent{Type: provider.EventToken, Text: "final output"})
	return &provider.CompletionResponse{Content: "final output"}, nil
}

func TestStreamProviderDoesNotReplayFailedAttemptOutput(t *testing.T) {
	var visible strings.Builder
	session := &Session{Provider: &partialRetryProvider{}}
	_, err := session.streamProvider(context.Background(), &provider.CompletionRequest{}, func(event provider.StreamEvent) error {
		visible.WriteString(event.Text)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := visible.String(), "final output"; got != want {
		t.Fatalf("visible stream = %q, want %q", got, want)
	}
}

type loopTestTool struct{}

func (loopTestTool) Name() string        { return "loop_test_tool" }
func (loopTestTool) Description() string { return "test tool" }
func (loopTestTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (loopTestTool) Execute(context.Context, map[string]interface{}) (string, error) {
	return "ok", nil
}

func TestExecuteToolCallFailsClosedWithoutApprovalUI(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AutoApprove = false
	tool := &countingTool{}
	registry := tools.NewRegistry()
	registry.Register(tool)
	session := &Session{Config: cfg, Tools: registry}

	result := session.executeToolCall(context.Background(), &provider.ToolCall{
		ID: "approval-test", Name: "counting_tool", Arguments: map[string]interface{}{},
	})
	if !result.IsError || !strings.Contains(result.Content, "approval") {
		t.Fatalf("result = %+v, want fail-closed approval error", result)
	}
	if tool.calls != 0 {
		t.Fatalf("tool calls = %d, want 0", tool.calls)
	}
}

type countingTool struct {
	calls int
}

func (t *countingTool) Name() string        { return "counting_tool" }
func (t *countingTool) Description() string { return "test tool" }
func (t *countingTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (t *countingTool) Execute(context.Context, map[string]interface{}) (string, error) {
	t.calls++
	return "executed", nil
}

func TestProcessUserInputStopsAtAgentTurnLimit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AutoApprove = true
	cfg.SetSetting("max_agent_turns", "3")
	looping := &loopingProvider{}
	registry := tools.NewRegistry()
	registry.Register(loopTestTool{})
	session := &Session{
		WorkspaceDir: t.TempDir(),
		Config:       cfg,
		Provider:     looping,
		Tools:        registry,
	}

	err := session.ProcessUserInput(context.Background(), "keep going")
	if err == nil || !strings.Contains(err.Error(), "maximum of 3 iterations") {
		t.Fatalf("ProcessUserInput() error = %v, want maximum-iterations error", err)
	}
	if looping.calls != 3 {
		t.Fatalf("provider calls = %d, want 3", looping.calls)
	}
}

func TestAgentTurnLimitUsesSafeDefaultAndBounds(t *testing.T) {
	if got, want := agentTurnLimit(nil), 25; got != want {
		t.Fatalf("nil config limit = %d, want %d", got, want)
	}

	cases := []struct {
		name  string
		value string
		want  int
	}{
		{name: "configured", value: "40", want: 40},
		{name: "zero falls back", value: "0", want: 25},
		{name: "negative falls back", value: "-2", want: 25},
		{name: "invalid falls back", value: "oops", want: 25},
		{name: "upper bound", value: "1000", want: 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.SetSetting("max_agent_turns", tc.value)
			if got := agentTurnLimit(cfg); got != tc.want {
				t.Fatalf("agentTurnLimit() = %d, want %d", got, tc.want)
			}
		})
	}
}
