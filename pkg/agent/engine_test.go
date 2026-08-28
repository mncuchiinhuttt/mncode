package agent

import (
	"context"
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
