package evals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mncode/pkg/tools"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RunEditBenchmark executes isolated edit cases in parallel using the real
// hash-aware EditTool. Every case gets a fresh temporary workspace.
func RunEditBenchmark(ctx context.Context, cases []EditCase, workers int) EditBenchmarkSummary {
	started := time.Now().UTC()
	if workers <= 0 {
		workers = 1
	}
	if workers > len(cases) && len(cases) > 0 {
		workers = len(cases)
	}
	results := make([]EditCaseResult, len(cases))
	jobs := make(chan int)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range jobs {
				results[index] = runEditCase(ctx, cases[index])
			}
		}()
	}
	for index := range cases {
		select {
		case jobs <- index:
		case <-ctx.Done():
			results[index] = EditCaseResult{Name: cases[index].Name, Error: ctx.Err().Error()}
		}
	}
	close(jobs)
	group.Wait()

	sort.SliceStable(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	summary := EditBenchmarkSummary{StartedAt: started, FinishedAt: time.Now().UTC(), Total: len(results), Cases: results}
	for _, result := range results {
		if result.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
		if result.StaleRejected {
			summary.StaleRejected++
		}
		if result.UnexpectedChange {
			summary.UnexpectedChanges++
		}
	}
	return summary
}

func runEditCase(ctx context.Context, testCase EditCase) (result EditCaseResult) {
	started := time.Now()
	result = EditCaseResult{Name: testCase.Name}
	defer func() { result.Duration = time.Since(started) }()
	if err := validateEditCase(testCase); err != nil {
		result.Error = err.Error()
		return result
	}
	workspace, err := os.MkdirTemp("", "mncode-edit-eval-")
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer os.RemoveAll(workspace)
	path := filepath.Join(workspace, testCase.Filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		result.Error = err.Error()
		return result
	}
	if err := os.WriteFile(path, []byte(testCase.Before), 0o600); err != nil {
		result.Error = err.Error()
		return result
	}
	tool := &tools.EditTool{BaseDir: workspace}
	if testCase.ExpectStaleRejection {
		staleHash := fingerprint([]byte(testCase.Before))
		if err := os.WriteFile(path, []byte(testCase.StaleContent), 0o600); err != nil {
			result.Error = err.Error()
			return result
		}
		_, err := tool.Execute(ctx, editArgs(testCase, staleHash))
		if err == nil || !strings.Contains(err.Error(), "stale edit rejected") {
			result.Error = fmt.Sprintf("expected stale rejection, got %v", err)
			return result
		}
		result.StaleRejected = true
	}
	current, err := os.ReadFile(path)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	_, err = tool.Execute(ctx, editArgs(testCase, fingerprint(current)))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if string(actual) != testCase.Expected {
		result.UnexpectedChange = true
		result.Error = "final file differs from expected content"
		return result
	}
	result.Passed = true
	return result
}

func validateEditCase(testCase EditCase) error {
	if strings.TrimSpace(testCase.Name) == "" {
		return fmt.Errorf("case name is required")
	}
	if testCase.Filename == "" || filepath.IsAbs(testCase.Filename) || strings.Contains(testCase.Filename, "..") {
		return fmt.Errorf("case filename must be a relative safe path")
	}
	if testCase.Target == "" {
		return fmt.Errorf("case target is required")
	}
	if testCase.ExpectStaleRejection && testCase.StaleContent == "" {
		return fmt.Errorf("stale content is required when stale rejection is enabled")
	}
	return nil
}

func editArgs(testCase EditCase, hash string) map[string]interface{} {
	return map[string]interface{}{
		"TargetFile":         testCase.Filename,
		"FileHash":           hash,
		"TargetContent":      testCase.Target,
		"ReplacementContent": testCase.Replacement,
		"AllowMultiple":      testCase.AllowMultiple,
	}
}

func fingerprint(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
