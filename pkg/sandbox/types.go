package sandbox

import "time"

// Fixture is a bounded, argv-based test scenario stored in the workspace.
type Fixture struct {
	SchemaVersion  int               `json:"schema_version"`
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Root           string            `json:"root"`
	Command        []string          `json:"command"`
	Env            map[string]string `json:"env,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	MaxOutputBytes int64             `json:"max_output_bytes,omitempty"`
}

// RunRequest selects a fixture and optional positional arguments.
type RunRequest struct {
	FixtureID string
	Args      []string
	Keep      bool
}

// RunResult records a fixture execution without source mutation.
type RunResult struct {
	SchemaVersion int       `json:"schema_version"`
	ID            string    `json:"id"`
	FixtureID     string    `json:"fixture_id"`
	Workspace     string    `json:"workspace"`
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at"`
	ExitCode      int       `json:"exit_code"`
	TimedOut      bool      `json:"timed_out"`
	Truncated     bool      `json:"truncated"`
	Stdout        string    `json:"stdout,omitempty"`
	Stderr        string    `json:"stderr,omitempty"`
	Error         string    `json:"error,omitempty"`
}
