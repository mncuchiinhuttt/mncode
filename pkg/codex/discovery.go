package codex

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RuntimeInfo describes the discovered Codex executable.
type RuntimeInfo struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

// DiscoverRuntime checks for official codex binary in PATH and verifies version.
func DiscoverRuntime(ctx context.Context) (*RuntimeInfo, error) {
	path, err := exec.LookPath("codex")
	if err != nil {
		return nil, fmt.Errorf("%w: lookpath failed", ErrRuntimeNotFound)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, path, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query version (%v)", ErrRuntimeNotFound, err)
	}

	versionStr := strings.TrimSpace(string(out))
	if versionStr == "" {
		versionStr = "unknown"
	}

	return &RuntimeInfo{
		Path:    path,
		Version: versionStr,
	}, nil
}
