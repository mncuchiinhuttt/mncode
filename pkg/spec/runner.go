package spec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"mncode/pkg/commandutil"
	"mncode/pkg/sandbox"
)

// Check evaluates only declared deterministic cases and never edits source.
func (s *Store) Check(ctx context.Context, contract Contract) (Matrix, error) {
	if s == nil {
		return Matrix{}, errors.New("spec store is required")
	}
	if err := Validate(contract); err != nil {
		return Matrix{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	matrix := Matrix{ContractID: contract.ID, GeneratedAt: time.Now().UTC(), Results: make([]CaseResult, 0, len(contract.Cases))}
	harness, err := sandbox.New(s.Workspace.Root)
	if err != nil {
		return matrix, err
	}
	for _, testCase := range contract.Cases {
		if err := ctx.Err(); err != nil {
			return matrix, err
		}
		result := s.checkCase(ctx, harness, testCase)
		matrix.Results = append(matrix.Results, result)
		switch result.Status {
		case StatusPass:
			matrix.Passed++
		case StatusFail:
			matrix.Failed++
		case StatusSkipped:
			matrix.Skipped++
		case StatusInvalid:
			matrix.Invalid++
		}
	}
	sort.Slice(matrix.Results, func(i, j int) bool { return matrix.Results[i].CaseID < matrix.Results[j].CaseID })
	return matrix, nil
}

func (s *Store) checkCase(ctx context.Context, harness *sandbox.Harness, testCase Case) (result CaseResult) {
	started := time.Now()
	result = CaseResult{CaseID: testCase.ID, Status: StatusInvalid}
	defer func() { result.Duration = time.Since(started) }()
	switch testCase.Kind {
	case "command":
		limits := s.Limits
		if limits.Timeout <= 0 {
			limits.Timeout = 30 * time.Second
		}
		if limits.MaxOutputBytes <= 0 {
			limits.MaxOutputBytes = 256 * 1024
		}
		stdout, stderr, runErr := harness.RunCommand(ctx, testCase.Command, nil, limits)
		var expected struct {
			ExitCode       *int   `json:"exit_code"`
			StdoutContains string `json:"stdout_contains"`
			StderrContains string `json:"stderr_contains"`
		}
		if len(testCase.Expected) > 0 && json.Unmarshal(testCase.Expected, &expected) != nil {
			result.Message = "command expected must be an object"
			return result
		}
		result.Actual = commandutil.Scrub(string(stdout) + string(stderr))
		var exitErr *exec.ExitError
		if runErr != nil && !errors.As(runErr, &exitErr) {
			if errors.Is(runErr, exec.ErrNotFound) || strings.Contains(runErr.Error(), "executable file not found") {
				result.Status, result.Message = StatusSkipped, "command executable is unavailable"
			} else {
				result.Status, result.Message = StatusFail, runErr.Error()
			}
			return result
		}
		exitCode := 0
		if exitErr != nil {
			exitCode = exitErr.ExitCode()
		}
		expectedCode := 0
		if expected.ExitCode != nil {
			expectedCode = *expected.ExitCode
		}
		if exitCode != expectedCode {
			result.Status, result.Message = StatusFail, fmt.Sprintf("expected exit code %d, got %d", expectedCode, exitCode)
			return result
		}
		if expected.StdoutContains != "" && !strings.Contains(string(stdout), expected.StdoutContains) {
			result.Status, result.Message = StatusFail, "stdout did not contain expected text"
			return result
		}
		if expected.StderrContains != "" && !strings.Contains(string(stderr), expected.StderrContains) {
			result.Status, result.Message = StatusFail, "stderr did not contain expected text"
			return result
		}
		result.Status = StatusPass
		return result
	case "file_exists", "file_contains":
		return s.checkFileCase(ctx, testCase)
	case "invariant":
		return checkInvariant(ctx, testCase, s.Workspace.Root)
	default:
		result.Message = "unsupported case kind"
		return result
	}
}
