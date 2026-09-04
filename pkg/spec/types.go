package spec

import (
	"context"
	"encoding/json"
	"time"
)

// Contract is a versioned, machine-checkable feature contract.
type Contract struct {
	SchemaVersion int         `json:"schema_version"`
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	Version       int         `json:"version"`
	Invariants    []Invariant `json:"invariants"`
	Cases         []Case      `json:"cases"`
	CreatedAt     time.Time   `json:"created_at"`
}

// Invariant describes a deterministic property of the contract.
type Invariant struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	Kind        string          `json:"kind"`
	Value       json.RawMessage `json:"value"`
}

// Case is one bounded observable check.
type Case struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Kind     string          `json:"kind"`
	Input    json.RawMessage `json:"input,omitempty"`
	Expected json.RawMessage `json:"expected,omitempty"`
	Command  []string        `json:"command,omitempty"`
	Tags     []string        `json:"tags,omitempty"`
}

// CaseStatus is the result category of one contract case.
type CaseStatus string

const (
	StatusPass    CaseStatus = "pass"
	StatusFail    CaseStatus = "fail"
	StatusSkipped CaseStatus = "skipped"
	StatusInvalid CaseStatus = "invalid"
)

// CaseResult records one deterministic case evaluation.
type CaseResult struct {
	CaseID   string        `json:"case_id"`
	Status   CaseStatus    `json:"status"`
	Duration time.Duration `json:"duration"`
	Actual   string        `json:"actual,omitempty"`
	Message  string        `json:"message,omitempty"`
}

// Matrix summarizes a contract check in stable case order.
type Matrix struct {
	ContractID  string       `json:"contract_id"`
	GeneratedAt time.Time    `json:"generated_at"`
	Results     []CaseResult `json:"results"`
	Passed      int          `json:"passed"`
	Failed      int          `json:"failed"`
	Skipped     int          `json:"skipped"`
	Invalid     int          `json:"invalid"`
}

// Runner is an optional custom evaluator for embedding applications.
type Runner interface {
	Run(context.Context, Contract) (Matrix, error)
}
