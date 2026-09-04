package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mncode/pkg/commandutil"
)

var allowedExecutables = map[string]bool{"go": true, "cargo": true, "rustc": true, "node": true, "bun": true, "deno": true, "python": true, "python3": true, "pytest": true, "ruby": true, "npm": true, "pnpm": true, "yarn": true, "echo": true, "printf": true}

// Run copies a fixture into a private run directory and executes fixed argv.
func (h *Harness) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	fixture, err := h.Load(ctx, req.FixtureID)
	if err != nil {
		return RunResult{}, err
	}
	if err := validateArgs(req.Args); err != nil {
		return RunResult{}, err
	}
	command := append(append([]string(nil), fixture.Command...), req.Args...)
	if err := validateCommand(command); err != nil {
		return RunResult{}, err
	}
	if err := validateCommandPaths(h.Workspace.Root, command); err != nil {
		return RunResult{}, err
	}
	id := commandutil.NewID("sandbox")
	if err := h.Workspace.RejectSymlinkPath(filepath.Join(h.RunDir, id)); err != nil {
		return RunResult{}, err
	}
	runRoot, err := h.path(filepath.Join(h.RunDir, id, "workspace"))
	if err != nil {
		return RunResult{}, err
	}
	if err := os.MkdirAll(runRoot, 0o700); err != nil {
		return RunResult{}, err
	}
	fixtureRelative := filepath.Join(h.FixtureDir, fixture.ID)
	if err := h.Workspace.RejectSymlinkPath(fixtureRelative); err != nil {
		return RunResult{}, err
	}
	fixtureRoot, err := h.path(fixtureRelative)
	if err != nil {
		return RunResult{}, err
	}
	limits := normalizedLimits(h.Limits)
	if err := copyFixture(fixtureRoot, runRoot, limits); err != nil {
		return RunResult{}, err
	}
	if fixture.TimeoutSeconds > 0 {
		limits.Timeout = time.Duration(fixture.TimeoutSeconds) * time.Second
	}
	if fixture.MaxOutputBytes > 0 && fixture.MaxOutputBytes < limits.MaxOutputBytes {
		limits.MaxOutputBytes = fixture.MaxOutputBytes
	}
	started := time.Now().UTC()
	stdout, stderr, runErr := commandutil.RunBoundedEnv(ctx, runRoot, command, limits, fixture.Env)
	result := RunResult{SchemaVersion: 1, ID: id, FixtureID: fixture.ID, Workspace: filepath.ToSlash(filepath.Join(h.RunDir, id, "workspace")), StartedAt: started, EndedAt: time.Now().UTC(), Stdout: commandutil.Scrub(string(stdout)), Stderr: commandutil.Scrub(string(stderr))}
	result.TimedOut = errors.Is(runErr, context.DeadlineExceeded)
	result.Truncated = errors.Is(runErr, commandutil.ErrOutputLimit)
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else if runErr != nil {
		result.ExitCode = -1
	}
	if runErr != nil && !result.Truncated {
		result.Error = commandutil.Scrub(runErr.Error())
	}
	if !req.Keep {
		_ = os.RemoveAll(filepath.Dir(runRoot))
	}
	manifestDir, err := h.path(filepath.Join(h.RunDir, id))
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(manifestDir, 0o700); err != nil {
		return result, err
	}
	if err := commandutil.WritePrivateJSON(filepath.Join(manifestDir, "manifest.json"), result); err != nil {
		return result, err
	}
	return result, nil
}

func validateFixture(fixture Fixture) error {
	if fixture.Root != "" && fixture.Root != "." {
		return fmt.Errorf("%w: fixture root must be '.'", errFixtureInvalid)
	}
	if err := validateCommand(fixture.Command); err != nil {
		return err
	}
	if fixture.TimeoutSeconds < 0 || fixture.TimeoutSeconds > 300 {
		return fmt.Errorf("%w: timeout must be 0-300 seconds", errFixtureInvalid)
	}
	return validateEnv(fixture.Env)
}

func validateCommand(command []string) error {
	if len(command) == 0 || len(command) > 32 {
		return fmt.Errorf("%w: command argv is empty or too long", errFixtureInvalid)
	}
	name := filepath.Base(command[0])
	if command[0] != name || !allowedExecutables[name] {
		return fmt.Errorf("%w: executable %q is not allowed", errFixtureInvalid, command[0])
	}
	return validateArgs(command[1:])
}

func validateArgs(args []string) error {
	if len(args) > 31 {
		return fmt.Errorf("%w: too many command arguments", errFixtureInvalid)
	}
	for _, arg := range args {
		if strings.IndexByte(arg, 0) >= 0 || strings.ContainsAny(arg, ";&|$<>\n\r") || interpreterEvalFlag(arg) {
			return fmt.Errorf("%w: shell or inline interpreter syntax is not allowed in argv", errFixtureInvalid)
		}
	}
	return nil
}
