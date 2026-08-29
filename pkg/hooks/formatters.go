package hooks

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SilentAutoFormat runs language-appropriate code formatters immediately after an edit
// without burning extra LLM turns or modifying prompt tokens.
func SilentAutoFormat(ctx context.Context, filePath string) error {
	if filePath == "" {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	switch ext {
	case ".go":
		// Go standard formatter
		cmd := exec.CommandContext(timeoutCtx, "gofmt", "-w", filePath)
		_ = cmd.Run()

	case ".ts", ".tsx", ".js", ".jsx", ".json", ".css", ".scss":
		// Prettier if available in local node_modules or global PATH
		cmd := exec.CommandContext(timeoutCtx, "npx", "--no-install", "prettier", "--write", filePath)
		_ = cmd.Run()

	case ".py":
		// Black / Ruff if available
		cmd := exec.CommandContext(timeoutCtx, "ruff", "format", filePath)
		if err := cmd.Run(); err != nil {
			cmd2 := exec.CommandContext(timeoutCtx, "black", "-q", filePath)
			_ = cmd2.Run()
		}

	case ".rs":
		// Rustfmt
		cmd := exec.CommandContext(timeoutCtx, "rustfmt", filePath)
		_ = cmd.Run()
	}

	return nil
}

// OnPostEdit performs both silent language formatting and custom user hook execution.
func OnPostEdit(ctx context.Context, workspaceDir, filePath string) {
	_ = SilentAutoFormat(ctx, filePath)
	_ = RunHook(ctx, workspaceDir, EventPostEdit, filePath)
}
