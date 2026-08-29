package orchestration

import (
	"encoding/json"
	"errors"
	"fmt"
	"mncode/pkg/persistence"
	"time"
)

// RunState represents the deterministic lifecycle state of a run.
type RunState string

const (
	StateQueued    RunState = "queued"
	StateRunning   RunState = "running"
	StateWaiting   RunState = "waiting"
	StateCompleted RunState = "completed"
	StateFailed    RunState = "failed"
	StateCancelled RunState = "cancelled"
	StateExpired   RunState = "expired"
)

var (
	ErrRunNotFound       = errors.New("run not found")
	ErrIllegalTransition = errors.New("illegal run state transition")
	ErrRunCancelled       = errors.New("run was cancelled")
	ErrRunTimeout         = errors.New("run timed out")
)

// IsTerminal reports whether a state is final.
func (s RunState) IsTerminal() bool {
	switch s {
	case StateCompleted, StateFailed, StateCancelled, StateExpired:
		return true
	default:
		return false
	}
}

// CanTransition validates lifecycle state transitions.
func CanTransition(from, to RunState) bool {
	if from == to {
		return true
	}
	switch from {
	case StateQueued:
		return to == StateRunning || to == StateCancelled
	case StateRunning:
		return to == StateWaiting || to == StateCompleted || to == StateFailed || to == StateCancelled
	case StateWaiting:
		return to == StateRunning || to == StateFailed || to == StateCancelled || to == StateExpired
	default:
		// Terminal states cannot transition further
		return false
	}
}

// RunKind classifies the workload.
type RunKind string

const (
	KindForegroundTurn RunKind = "foreground_turn"
	KindSubagent       RunKind = "subagent"
	KindProcess        RunKind = "process"
	KindAutomation     RunKind = "automation"
	KindRemote         RunKind = "remote"
)

// RunMeta holds immutable identity and ownership metadata.
type RunMeta struct {
	ID           string                 `json:"id"`
	ChatID       string                 `json:"chatId,omitempty"`
	Generation   int64                  `json:"generation,omitempty"`
	ParentRunID  string                 `json:"parentRunId,omitempty"`
	Owner        string                 `json:"owner,omitempty"`
	WorkspaceDir string                 `json:"workspaceDir,omitempty"`
	Kind         RunKind                `json:"kind"`
	Model        string                 `json:"model,omitempty"`
	Provider     string                 `json:"provider,omitempty"`
	Labels       map[string]string      `json:"labels,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// EventEnvelope is an observable event emitted by a run.
type EventEnvelope struct {
	ID         int64           `json:"id,omitempty"`
	RunID      string          `json:"runId"`
	JobID      string          `json:"jobId,omitempty"`
	ChatID     string          `json:"chatId,omitempty"`
	Generation int64           `json:"generation,omitempty"`
	Sequence   int             `json:"sequence"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

// RunSnapshot is an immutable view of a run at a point in time.
type RunSnapshot struct {
	Meta       RunMeta    `json:"meta"`
	State      RunState   `json:"state"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Error      string     `json:"error,omitempty"`
	Result     string     `json:"result,omitempty"`
	ExitCode   int        `json:"exitCode,omitempty"`
	TokensIn   int        `json:"tokensIn,omitempty"`
	TokensOut  int        `json:"tokensOut,omitempty"`
}

// ToPersistence converts a RunSnapshot to a persistence.RunRecord.
func (s RunSnapshot) ToPersistence() persistence.RunRecord {
	metaJSON, _ := json.Marshal(s.Meta)
	return persistence.RunRecord{
		ID:         s.Meta.ID,
		SessionID:  s.Meta.ChatID,
		Profile:    persistence.Profile{ProfileID: "default", WorkspaceDir: s.Meta.WorkspaceDir, ChatID: s.Meta.ChatID},
		Status:     string(s.State),
		Model:      s.Meta.Model,
		Provider:   s.Meta.Provider,
		StartedAt:  s.StartedAt,
		FinishedAt: s.FinishedAt,
		Metadata:   metaJSON,
	}
}

// Validate validates that a RunMeta has required fields.
func (m RunMeta) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("run ID is required")
	}
	if m.Kind == "" {
		return fmt.Errorf("run kind is required")
	}
	return nil
}
