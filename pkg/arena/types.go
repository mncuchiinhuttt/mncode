package arena

import (
	"context"
	"time"
)

// Source identifies the bounded diff under review.
type Source struct {
	Base         string   `json:"base,omitempty"`
	Head         string   `json:"head,omitempty"`
	Diff         string   `json:"-"`
	DiffSHA256   string   `json:"diff_sha256"`
	ChangedFiles []string `json:"changed_files"`
	RepoRoot     string   `json:"repo_root"`
}

// Options bounds reviewer fan-out and diff collection.
type Options struct {
	Models           []string
	Rounds           int
	MaxDiffBytes     int64
	Timeout          time.Duration
	IncludeUntracked bool
}

// Finding is an evidence-backed advisory from one reviewer role.
type Finding struct {
	ID             string  `json:"id"`
	Severity       string  `json:"severity"`
	Category       string  `json:"category"`
	File           string  `json:"file"`
	Line           int     `json:"line,omitempty"`
	Evidence       string  `json:"evidence"`
	Impact         string  `json:"impact"`
	Recommendation string  `json:"recommendation"`
	Confidence     float64 `json:"confidence"`
}

// Report is a deterministic risk report for one source diff.
type Report struct {
	SchemaVersion int       `json:"schema_version"`
	ID            string    `json:"id"`
	Source        Source    `json:"source"`
	Findings      []Finding `json:"findings"`
	Verdict       string    `json:"verdict"`
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at"`
}

// Reviewer performs one role-specific review without changing the workspace.
type Reviewer interface {
	Review(ctx context.Context, source Source, role string) ([]Finding, error)
}
