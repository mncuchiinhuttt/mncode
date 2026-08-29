package evals

import (
	"context"
	"testing"
)

func TestRunEditBenchmarkParallelAndMeasuresStaleRejection(t *testing.T) {
	cases := []EditCase{
		{
			Name: "replace greeting", Filename: "greet.go",
			Before: "package main\n\nfunc greet() string { return \"hi\" }\n",
			Target: "\"hi\"", Replacement: "\"hello\"",
			Expected: "package main\n\nfunc greet() string { return \"hello\" }\n",
		},
		{
			Name: "reject stale then edit", Filename: "nested/config.txt",
			Before:       "mode=dev\n",
			StaleContent: "mode=dev\nexternal=true\n",
			Target:       "mode=dev", Replacement: "mode=prod",
			Expected: "mode=prod\nexternal=true\n", ExpectStaleRejection: true,
		},
	}
	summary := RunEditBenchmark(context.Background(), cases, 2)
	if summary.Total != 2 || summary.Passed != 2 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.StaleRejected != 1 || summary.Reliability() != 1 {
		t.Fatalf("unexpected reliability metrics: %+v", summary)
	}
	for _, result := range summary.Cases {
		if result.Duration <= 0 {
			t.Fatalf("case duration was not recorded: %+v", result)
		}
	}
}

func TestRunEditBenchmarkRejectsUnsafeCase(t *testing.T) {
	summary := RunEditBenchmark(context.Background(), []EditCase{{Name: "unsafe", Filename: "../outside", Target: "x"}}, 1)
	if summary.Passed != 0 || summary.Failed != 1 || summary.Cases[0].Error == "" {
		t.Fatalf("expected failed unsafe case, got %+v", summary)
	}
}
