package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mncode/pkg/hub"
)

type HookEvent string

const (
	EventPostEdit  HookEvent = "post_edit"
	EventPreCommit HookEvent = "pre_commit"
	EventOnError   HookEvent = "on_error"
)

// RunHook executes custom user hook scripts in <workspace>/.mncode/hooks/<event>.<sh|py|js>.
func RunHook(ctx context.Context, workspaceDir string, event HookEvent, targetFile string) error {
	if workspaceDir == "" {
		workspaceDir = "."
	}
	hooksDir := filepath.Join(workspaceDir, ".mncode", "hooks")
	if _, err := os.Stat(hooksDir); os.IsNotExist(err) {
		return nil
	}

	extensions := []string{".sh", ".py", ".js", ""}
	var hookScript string

	for _, ext := range extensions {
		candidate := filepath.Join(hooksDir, string(event)+ext)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			hookScript = candidate
			break
		}
	}

	if hookScript == "" {
		return nil
	}

	timeout := 10 * time.Second
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if strings.HasSuffix(hookScript, ".py") {
		cmd = exec.CommandContext(execCtx, "python3", hookScript, targetFile)
	} else if strings.HasSuffix(hookScript, ".js") {
		cmd = exec.CommandContext(execCtx, "node", hookScript, targetFile)
	} else {
		cmd = exec.CommandContext(execCtx, "sh", hookScript, targetFile)
	}

	cmd.Dir = workspaceDir
	cmd.Env = hub.SanitizeProcessEnv(map[string]string{
		"MNCODE_EVENT":       string(event),
		"MNCODE_TARGET_FILE": targetFile,
		"MNCODE_WORKSPACE":   workspaceDir,
	})

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hook %s failed: %w\n%s", event, err, string(output))
	}
	return nil
}
