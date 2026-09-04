package replay

import (
	"encoding/json"
	"time"

	"mncode/pkg/commandutil"
	"mncode/pkg/provider"
)

// Kind identifies an engine lifecycle event in a trace.
type Kind string

const (
	KindSessionStart     Kind = "session_start"
	KindPrompt           Kind = "prompt"
	KindProviderReq      Kind = "provider_request"
	KindProviderResponse Kind = "provider_response"
	KindThinking         Kind = "thinking"
	KindToolCall         Kind = "tool_call"
	KindToolResult       Kind = "tool_result"
	KindTurnEnd          Kind = "turn_end"
	KindError            Kind = "error"
	KindMessage          Kind = "message"
	KindSessionEnd       Kind = "session_end"
)

// Event is one ordered, redacted lifecycle record.
type Event struct {
	Seq  int64           `json:"seq"`
	Kind Kind            `json:"kind"`
	At   time.Time       `json:"at"`
	Turn int             `json:"turn,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Trace describes one persisted flight recorder stream.
type Trace struct {
	SchemaVersion int       `json:"schema_version"`
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	WorkspaceRoot string    `json:"workspace_root"`
	WorkspaceID   string    `json:"workspace_id"`
	Model         string    `json:"model,omitempty"`
	Provider      string    `json:"provider,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at,omitempty"`
	Events        int       `json:"events"`
	Complete      bool      `json:"complete"`
	Checksum      string    `json:"checksum,omitempty"`
}

// ForkRequest selects a trace prefix; tool effects are never replayed.
type ForkRequest struct {
	TraceID      string
	At           int64
	NewSessionID string
	Name         string
	ReplayTools  bool
}

// ForkResult contains reconstructed conversation context for a new session.
type ForkResult struct {
	SessionID     string
	ParentTraceID string
	At            int64
	History       []provider.Message
	Source        Trace
}

// Store owns traces under one canonical workspace.
type Store struct {
	Workspace commandutil.Workspace
	Dir       string
	MaxEvents int
	MaxBytes  int64
}
