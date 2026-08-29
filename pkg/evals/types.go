// Package evals provides deterministic, isolated harness evaluations for mncode.
package evals

import "time"

// EditCase describes one real file-edit scenario evaluated in a temporary workspace.
type EditCase struct {
	Name                 string `json:"name"`
	Filename             string `json:"filename"`
	Before               string `json:"before"`
	StaleContent         string `json:"staleContent,omitempty"`
	Target               string `json:"target"`
	Replacement          string `json:"replacement"`
	Expected             string `json:"expected"`
	AllowMultiple        bool   `json:"allowMultiple,omitempty"`
	ExpectStaleRejection bool   `json:"expectStaleRejection,omitempty"`
}

// EditCaseResult records observable correctness outcomes for one case.
type EditCaseResult struct {
	Name             string        `json:"name"`
	Passed           bool          `json:"passed"`
	StaleRejected    bool          `json:"staleRejected"`
	UnexpectedChange bool          `json:"unexpectedChange"`
	Error            string        `json:"error,omitempty"`
	Duration         time.Duration `json:"duration"`
}

// EditBenchmarkSummary aggregates edit reliability metrics.
type EditBenchmarkSummary struct {
	StartedAt         time.Time        `json:"startedAt"`
	FinishedAt        time.Time        `json:"finishedAt"`
	Total             int              `json:"total"`
	Passed            int              `json:"passed"`
	Failed            int              `json:"failed"`
	StaleRejected     int              `json:"staleRejected"`
	UnexpectedChanges int              `json:"unexpectedChanges"`
	Cases             []EditCaseResult `json:"cases"`
}

// Reliability returns the fraction of cases that produced the exact expected file.
func (s EditBenchmarkSummary) Reliability() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Passed) / float64(s.Total)
}
