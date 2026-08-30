package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mncode/pkg/commandutil"
)

func interpreterEvalFlag(arg string) bool {
	norm := strings.ToLower(strings.TrimSpace(arg))
	if norm == "" {
		return false
	}
	if norm == "-c" || strings.HasPrefix(norm, "-c=") || (strings.HasPrefix(norm, "-c") && len(norm) > 2 && norm[2] != '-') {
		return true
	}
	if norm == "-e" || strings.HasPrefix(norm, "-e=") || (strings.HasPrefix(norm, "-e") && len(norm) > 2 && norm[2] != '-') {
		return true
	}
	if norm == "--eval" || strings.HasPrefix(norm, "--eval=") || norm == "-eval" || strings.HasPrefix(norm, "-eval=") {
		return true
	}
	if norm == "--command" || strings.HasPrefix(norm, "--command=") || norm == "-command" || strings.HasPrefix(norm, "-command=") {
		return true
	}
	if norm == "eval" || norm == "exec" || strings.HasPrefix(norm, "--exec=") || strings.HasPrefix(norm, "-exec=") {
		return true
	}
	return false
}

func validateEnv(env map[string]string) error {
	for key, value := range env {
		lowerKey := strings.ToLower(key)
		secretName := strings.Contains(lowerKey, "key") || strings.Contains(lowerKey, "token") || strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "auth")
		if !safeEnvKey(key) || strings.ContainsAny(value, "\x00\n\r") || secretName {
			return fmt.Errorf("%w: unsafe environment value %q", errFixtureInvalid, key)
		}
	}
	return nil
}

func safeEnvKey(key string) bool {
	if key == "CI" || key == "NODE_ENV" || key == "RUST_BACKTRACE" {
		return true
	}
	if !strings.HasPrefix(key, "MNCODE_") || len(key) >= 64 {
		return false
	}
	for _, r := range key {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// RunCommand runs a validated command against a temporary workspace copy.
func (h *Harness) RunCommand(ctx context.Context, command []string, env map[string]string, limits commandutil.Limits) (stdout, stderr []byte, err error) {
	if err := validateCommand(command); err != nil {
		return nil, nil, err
	}
	if err := validateCommandPaths(h.Workspace.Root, command); err != nil {
		return nil, nil, err
	}
	if err := validateEnv(env); err != nil {
		return nil, nil, err
	}
	limits = normalizedLimits(limits)
	root, err := os.MkdirTemp("", "mncode-spec-run-*")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(root)
	if err := copyWorkspace(h.Workspace.Root, root, limits); err != nil {
		return nil, nil, err
	}
	return commandutil.RunBoundedEnv(ctx, root, command, limits, env)
}

func validateCommandPaths(workspace string, command []string) error {
	root := filepath.Clean(workspace)
	aliases := []string{filepath.ToSlash(root)}
	if strings.HasPrefix(root, string(filepath.Separator)+"private"+string(filepath.Separator)) {
		aliases = append(aliases, filepath.ToSlash(strings.TrimPrefix(root, string(filepath.Separator)+"private")))
	}
	for _, arg := range command[1:] {
		normalized := filepath.ToSlash(arg)
		if filepath.IsAbs(arg) || embeddedAbsolutePath(normalized) || normalized == ".." || strings.HasPrefix(normalized, "../") || strings.Contains(normalized, "/../") {
			return fmt.Errorf("%w: command argument may not target the source workspace", errFixtureInvalid)
		}
		for _, alias := range aliases {
			if strings.Contains(normalized, alias) {
				return fmt.Errorf("%w: command argument may not target the source workspace", errFixtureInvalid)
			}
		}
	}
	return nil
}

func embeddedAbsolutePath(value string) bool {
	for _, marker := range []string{"('/", "(\"/", "=/", "= /"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func copyWorkspace(source, destination string, limits commandutil.Limits) error {
	count := 0
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		if entry.IsDir() {
			if skipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if secretFixturePath(rel) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > limits.MaxFileBytes {
			return fmt.Errorf("workspace file %q exceeds sandbox copy limit", rel)
		}
		count++
		if count > limits.MaxFiles {
			return fmt.Errorf("workspace exceeds %d sandbox files", limits.MaxFiles)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
}

func normalizedLimits(limits commandutil.Limits) commandutil.Limits {
	defaults := commandutil.DefaultLimits()
	if limits.Timeout <= 0 {
		limits.Timeout = defaults.Timeout
	}
	if limits.MaxFiles <= 0 {
		limits.MaxFiles = defaults.MaxFiles
	}
	if limits.MaxFileBytes <= 0 {
		limits.MaxFileBytes = defaults.MaxFileBytes
	}
	if limits.MaxOutputBytes <= 0 {
		limits.MaxOutputBytes = defaults.MaxOutputBytes
	}
	return limits
}

// ValidateCommand exposes the fixture command policy to other local harnesses.
func ValidateCommand(command []string) error { return validateCommand(command) }

// ValidateEnvironment exposes the safe fixture environment policy to other local harnesses.
func ValidateEnvironment(env map[string]string) error { return validateEnv(env) }

func skipDir(name string) bool {
	switch name {
	case ".git", ".mncode", "node_modules", "vendor", "dist", "build", "target", ".next":
		return true
	}
	return false
}
