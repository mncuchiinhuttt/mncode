package commandutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Limits bounds subprocesses and workspace scans.
type Limits struct {
	Timeout        time.Duration
	MaxOutputBytes int64
	MaxFiles       int
	MaxFileBytes   int64
}

// DefaultLimits returns conservative limits for local command features.
func DefaultLimits() Limits {
	return Limits{Timeout: 30 * time.Second, MaxOutputBytes: 256 * 1024, MaxFiles: 5000, MaxFileBytes: 2 * 1024 * 1024}
}

// ErrOutputLimit indicates that a subprocess exceeded its captured output cap.
var ErrOutputLimit = errors.New("subprocess output limit exceeded")

// RunBounded executes a fixed argv without a shell and with a sanitized environment.
func RunBounded(ctx context.Context, root string, argv []string, limits Limits) (stdout, stderr []byte, err error) {
	return RunBoundedEnv(ctx, root, argv, limits, nil)
}

// RunBoundedEnv is RunBounded with explicitly supplied safe environment values.
func RunBoundedEnv(ctx context.Context, root string, argv []string, limits Limits, extra map[string]string) (stdout, stderr []byte, err error) {
	if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
		return nil, nil, errors.New("command argv cannot be empty")
	}
	if err := validateEnvironment(extra); err != nil {
		return nil, nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if limits.Timeout <= 0 {
		limits.Timeout = DefaultLimits().Timeout
	}
	if limits.MaxOutputBytes <= 0 {
		limits.MaxOutputBytes = DefaultLimits().MaxOutputBytes
	}
	runCtx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()
	executable, err := resolveExecutable(argv[0])
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(runCtx, executable, argv[1:]...)
	cmd.Dir = root
	prepareProcess(cmd)
	cmd.Cancel = func() error { return killProcessTree(cmd) }
	cmd.WaitDelay = 500 * time.Millisecond
	cmd.Env = append(safeEnvironment(), formatEnvironment(extra)...)
	var out, errOut cappedBuffer
	out.limit, errOut.limit = limits.MaxOutputBytes, limits.MaxOutputBytes
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err = cmd.Run()
	stdout, stderr = out.Bytes(), errOut.Bytes()
	if out.truncated || errOut.truncated {
		return stdout, stderr, fmt.Errorf("%w: %s", ErrOutputLimit, argv[0])
	}
	if runCtx.Err() != nil {
		return stdout, stderr, runCtx.Err()
	}
	return stdout, stderr, err
}

func formatEnvironment(extra map[string]string) []string {
	if len(extra) == 0 {
		return nil
	}
	keys := make([]string, 0, len(extra))
	for key := range extra {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+extra[key])
	}
	return values
}
func validateEnvironment(extra map[string]string) error {
	for key, value := range extra {
		if !safeEnvironmentKey(key) || strings.ContainsAny(value, "\x00\n\r") {
			return fmt.Errorf("unsafe environment variable %q", key)
		}
	}
	return nil
}

func safeEnvironmentKey(key string) bool {
	switch key {
	case "CI", "NODE_ENV", "RUST_BACKTRACE":
		return true
	}
	if !strings.HasPrefix(key, "MNCODE_") || len(key) >= 64 {
		return false
	}
	for _, r := range key {
		if r != '_' && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	lower := strings.ToLower(key)
	return !strings.Contains(lower, "key") && !strings.Contains(lower, "token") &&
		!strings.Contains(lower, "secret") && !strings.Contains(lower, "password") &&
		!strings.Contains(lower, "auth")
}

func safeEnvironment() []string {
	return []string{"PATH=" + safePath(), "HOME=" + os.TempDir(), "LANG=C", "LC_ALL=C", "MNCODE_SANDBOX=1"}
}

func safePath() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows\System32;C:\Windows`
	}
	paths := []string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/local/go/bin", "/usr/bin", "/bin"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append([]string{
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".cargo", "bin"),
			filepath.Join(home, ".bun", "bin"),
			filepath.Join(home, "go", "bin"),
		}, paths...)
	}
	return strings.Join(paths, string(os.PathListSeparator))
}

func resolveExecutable(name string) (string, error) {
	if filepath.IsAbs(name) {
		return validateExecutablePath(name)
	}
	if strings.ContainsAny(name, `/\\`) {
		return "", fmt.Errorf("executable path must be a name, not %q", name)
	}
	for _, dir := range strings.Split(safePath(), string(os.PathListSeparator)) {
		candidate := filepath.Join(dir, name)
		if runtime.GOOS == "windows" && filepath.Ext(candidate) == "" {
			candidate += ".exe"
		}
		if resolved, err := validateExecutablePath(candidate); err == nil {
			return resolved, nil
		}
	}
	return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
}

func validateExecutablePath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("executable is not runnable: %s", path)
	}
	return path, nil
}
