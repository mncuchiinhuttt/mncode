package combos

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockExecutor struct {
	mu        sync.Mutex
	calls     []string
	failModel string
	failError error
}

func (m *mockExecutor) ExecuteMember(ctx context.Context, member ComboMember, model string, prompt string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, fmt.Sprintf("%s:%s", member.Role, model))

	if m.failModel != "" && model == m.failModel {
		return "", m.failError
	}
	return fmt.Sprintf("Output from %s using %s", member.Role, model), nil
}

type mockListener struct {
	mu        sync.Mutex
	fallbacks []string
}

func (l *mockListener) OnModelFallback(role, fromModel, toModel string, cause error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fallbacks = append(l.fallbacks, fmt.Sprintf("%s:%s->%s", role, fromModel, toModel))
}

func (l *mockListener) OnComboStart(comboID string, name string, mode ExecutionMode, memberCount int)     {}
func (l *mockListener) OnComboStepStart(comboID string, role string, model string, stepIndex int, total int) {}
func (l *mockListener) OnComboStepDone(comboID string, role string, modelUsed string, duration time.Duration, output string) {
}
func (l *mockListener) OnComboStepError(comboID string, role string, err error)              {}
func (l *mockListener) OnComboComplete(comboID string, duration time.Duration, output string) {}

func TestRunnerPipelineExecution(t *testing.T) {
	store, _ := NewStore("")
	exec := &mockExecutor{}
	listener := &mockListener{}
	runner := NewRunner(store, exec, listener)

	res, err := runner.Run(context.Background(), "feature-delivery", "Build login form")
	if err != nil {
		t.Fatalf("Pipeline run failed: %v", err)
	}
	if len(res.Steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(res.Steps))
	}
	if !strings.Contains(res.FinalOutput, "code-reviewer") {
		t.Fatalf("expected final output from reviewer, got %q", res.FinalOutput)
	}
}

func TestRunnerDebateExecution(t *testing.T) {
	store, _ := NewStore("")
	exec := &mockExecutor{}
	listener := &mockListener{}
	runner := NewRunner(store, exec, listener)

	res, err := runner.Run(context.Background(), "critic-refactor", "Refactor auth handler")
	if err != nil {
		t.Fatalf("Debate run failed: %v", err)
	}
	if len(res.Steps) == 0 {
		t.Fatal("expected debate steps, got 0")
	}
	if !strings.Contains(res.FinalOutput, "tester") && !strings.Contains(res.FinalOutput, "architect") {
		t.Fatalf("expected decider output, got %q", res.FinalOutput)
	}
}

func TestRunnerFanOutExecution(t *testing.T) {
	store, _ := NewStore("")
	exec := &mockExecutor{}
	listener := &mockListener{}
	runner := NewRunner(store, exec, listener)

	res, err := runner.Run(context.Background(), "security-audit", "Scan repository")
	if err != nil {
		t.Fatalf("FanOut run failed: %v", err)
	}
	if len(res.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(res.Steps))
	}
	if !strings.Contains(res.FinalOutput, "refactorer") {
		t.Fatalf("expected integrator output, got %q", res.FinalOutput)
	}
}

func TestRunnerAutomaticModelFallback(t *testing.T) {
	store, _ := NewStore("")
	combo := Combo{
		ID:   "test-fallback-combo",
		Name: "Test Fallback",
		Mode: ModePipeline,
		Members: []ComboMember{
			{Role: "planner", Model: "failing-model", FallbackModel: "backup-model"},
		},
	}
	_ = store.Save(combo)

	exec := &mockExecutor{
		failModel: "failing-model",
		failError: errors.New("HTTP 429: Rate limit exceeded"),
	}
	listener := &mockListener{}
	runner := NewRunner(store, exec, listener)

	res, err := runner.Run(context.Background(), "test-fallback-combo", "Plan task")
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if len(listener.fallbacks) != 1 {
		t.Fatalf("expected 1 fallback event, got %v", listener.fallbacks)
	}
	if res.Steps[0].ModelUsed != "backup-model" {
		t.Fatalf("expected modelUsed=backup-model, got %s", res.Steps[0].ModelUsed)
	}
}
