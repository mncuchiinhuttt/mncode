package spec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"mncode/pkg/tools"
)

func (s *Store) checkFileCase(ctx context.Context, testCase Case) (result CaseResult) {
	result = CaseResult{CaseID: testCase.ID, Status: StatusInvalid}
	var input struct {
		Path string `json:"path"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(testCase.Input, &input); err != nil || strings.TrimSpace(input.Path) == "" {
		result.Message = "file case input must include path"
		return result
	}
	expected, err := expectedBool(testCase.Expected, true)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	path, err := tools.ResolveWorkspacePath(s.Workspace.Root, input.Path, true)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			result.Message = err.Error()
			return result
		}
	}
	info, statErr := os.Stat(path)
	exists := statErr == nil
	if testCase.Kind == "file_exists" {
		result.Actual = fmt.Sprintf("exists=%t", exists)
		result.Status = statusBool(exists == expected)
		if result.Status == StatusFail {
			result.Message = "file existence did not match expected value"
		}
		return result
	}
	if statErr != nil || !info.Mode().IsRegular() {
		result.Status, result.Message = StatusFail, "file is unavailable"
		return result
	}
	file, err := os.Open(path)
	if err != nil {
		result.Status, result.Message = StatusFail, err.Error()
		return result
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 512*1024+1))
	if err != nil {
		result.Status, result.Message = StatusFail, err.Error()
		return result
	}
	if len(data) > 512*1024 {
		result.Status, result.Message = StatusInvalid, "file case exceeds 512KB"
		return result
	}
	contains := strings.Contains(string(data), input.Text)
	result.Actual = fmt.Sprintf("contains=%t", contains)
	result.Status = statusBool(contains == expected)
	if result.Status == StatusFail {
		result.Message = "file content did not match expected value"
	}
	return result
}

func checkInvariant(ctx context.Context, testCase Case, root string) CaseResult {
	result := CaseResult{CaseID: testCase.ID, Status: StatusInvalid}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			result.Message = err.Error()
			return result
		}
	}
	var input struct {
		Operator string `json:"operator"`
		Value    string `json:"value"`
	}
	if err := json.Unmarshal(testCase.Input, &input); err != nil {
		result.Message = err.Error()
		return result
	}
	expected, err := expectedBool(testCase.Expected, true)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	actual := false
	switch input.Operator {
	case "non_empty":
		actual = input.Value != ""
	case "path_within_workspace":
		_, err = tools.ResolveWorkspacePath(root, input.Value, false)
		actual = err == nil
	default:
		result.Message = "unsupported invariant"
		return result
	}
	result.Actual = fmt.Sprintf("value=%t", actual)
	result.Status = statusBool(actual == expected)
	if result.Status == StatusFail {
		result.Message = "invariant did not match expected value"
	}
	return result
}

func expectedBool(raw json.RawMessage, fallback bool) (bool, error) {
	if len(raw) == 0 {
		return fallback, nil
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, errors.New("expected must be a boolean")
	}
	return value, nil
}
func statusBool(ok bool) CaseStatus {
	if ok {
		return StatusPass
	}
	return StatusFail
}
