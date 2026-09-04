package selfheal

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ReproState represents the progress state of bug reproduction and healing.
type ReproState string

const (
	StateReproRed   ReproState = "repro_failing_red"   // Bug successfully reproduced with failing test
	StateReproGreen ReproState = "repro_passing_green" // Fix applied and reproduction test passes
	StateVerified   ReproState = "regression_verified" // Full test suite confirmed green
)

// ReproSession manages a single self-healing bug fix lifecycle.
type ReproSession struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	TestFile    string     `json:"testFile"`
	TestCommand string     `json:"testCommand"`
	State       ReproState `json:"state"`
	StartTime   time.Time  `json:"startTime"`
	EndTime     time.Time  `json:"endTime"`
}

// RunReproTest executes the reproduction test and reports whether it is currently failing.
func RunReproTest(ctx context.Context, workspaceDir, testCommand string) (bool, string, error) {
	if testCommand == "" {
		testCommand = "go test -count=1 ./..."
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parts := strings.Fields(testCommand)
	if len(parts) == 0 {
		return false, "", fmt.Errorf("empty test command")
	}

	cmd := exec.CommandContext(execCtx, parts[0], parts[1:]...)
	if workspaceDir != "" {
		cmd.Dir = workspaceDir
	}

	output, err := cmd.CombinedOutput()
	isFailing := err != nil
	return isFailing, string(output), nil
}
