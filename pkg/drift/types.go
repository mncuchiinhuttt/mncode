package drift

import (
	"time"

	"mncode/pkg/repomap"
)

// Severity controls whether a drift finding can block strict checks.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Layer assigns source paths to a named architectural boundary.
type Layer struct {
	Name  string   `json:"name"`
	Globs []string `json:"globs"`
}

// Policy controls baseline scanning and architectural rules.
type Policy struct {
	Ignore           []string            `json:"ignore,omitempty"`
	FailOn           []Severity          `json:"fail_on,omitempty"`
	MaxChangedFiles  int                 `json:"max_changed_files,omitempty"`
	MaxImportEdges   int                 `json:"max_import_edges,omitempty"`
	Layers           []Layer             `json:"layers,omitempty"`
	ForbiddenImports map[string][]string `json:"forbidden_imports,omitempty"`
	DenyCycles       bool                `json:"deny_cycles,omitempty"`
}

// FileSnapshot captures structure and identity without storing source bodies.
type FileSnapshot struct {
	Path    string           `json:"path"`
	SHA256  string           `json:"sha256"`
	Size    int64            `json:"size"`
	Symbols []repomap.Symbol `json:"symbols,omitempty"`
	Imports []string         `json:"imports,omitempty"`
}

// Baseline is the immutable architectural snapshot used for later checks.
type Baseline struct {
	SchemaVersion int            `json:"schema_version"`
	ID            string         `json:"id"`
	WorkspaceRoot string         `json:"workspace_root"`
	WorkspaceID   string         `json:"workspace_id"`
	ToolVersion   string         `json:"tool_version"`
	CreatedAt     time.Time      `json:"created_at"`
	Policy        Policy         `json:"policy"`
	Files         []FileSnapshot `json:"files"`
}

// Finding describes one deterministic structural change.
type Finding struct {
	Severity Severity    `json:"severity"`
	Code     string      `json:"code"`
	Path     string      `json:"path"`
	Message  string      `json:"message"`
	Before   interface{} `json:"before,omitempty"`
	After    interface{} `json:"after,omitempty"`
}

// Report is the result of comparing the current workspace against a baseline.
type Report struct {
	SchemaVersion int        `json:"schema_version"`
	BaselineID    string     `json:"baseline_id"`
	WorkspaceRoot string     `json:"workspace_root"`
	GeneratedAt   time.Time  `json:"generated_at"`
	ChangedFiles  int        `json:"changed_files"`
	FailOn        []Severity `json:"fail_on,omitempty"`
	Findings      []Finding  `json:"findings"`
	Drifted       bool       `json:"drifted"`
}

// ExitCode returns a shell-friendly result for a report.
func (r Report) ExitCode(strict bool) int {
	for _, finding := range r.Findings {
		if containsSeverity(r.FailOn, finding.Severity) || finding.Severity == SeverityError || strict && finding.Severity == SeverityWarning {
			return 1
		}
	}
	return 0
}

func containsSeverity(values []Severity, target Severity) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
