package arena

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"mncode/pkg/commandutil"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var diffSecretRE = regexp.MustCompile(`(?i)(["']?(?:api[_-]?key|access[_-]?token|authorization|bearer|token|secret|password|credentials?|private[_-]?key)["']?\s*[:=]\s*)(?:["']?)(?:bearer\s+)?[^\s"',;}\]]+`)
var authorizationRE = regexp.MustCompile(`(?i)(authorization\s*:\s*)(?:[A-Za-z]+\s+)?[^\s"',;}\]]+`)

// CollectSource obtains a bounded git diff without changing repository state.
func CollectSource(ctx context.Context, workspace, base, head string, includeUntracked bool, maxBytes int64) (Source, error) {
	workingTree := base == "" && head == ""
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return Source{}, err
	}
	if maxBytes <= 0 {
		maxBytes = 512 * 1024
	}
	if err := validateRef(base); err != nil {
		return Source{}, err
	}
	if err := validateRef(head); err != nil {
		return Source{}, err
	}
	if base != "" {
		if err := verifyRef(ctx, root.Root, base); err != nil {
			return Source{}, err
		}
	}
	if head != "" {
		if err := verifyRef(ctx, root.Root, head); err != nil {
			return Source{}, err
		}
	}
	args := []string{"git", "diff", "--no-ext-diff", "--unified=80"}
	if base != "" && head == "" {
		head = "HEAD"
	}
	if base == "" && head != "" {
		base = "HEAD"
	}
	if base != "" {
		args = append(args, base)
	}
	if head != "" {
		args = append(args, head)
	}
	if base == "" && head == "" {
		args = append(args, "HEAD")
	}
	limits := commandutil.DefaultLimits()
	limits.Timeout = 20 * time.Second
	limits.MaxOutputBytes = maxBytes
	stdout, stderr, err := commandutil.RunBounded(ctx, root.Root, args, limits)
	if err != nil && workingTree && strings.Contains(string(stderr), "unknown revision") {
		stdout, stderr, err = commandutil.RunBounded(ctx, root.Root, args[:len(args)-1], limits)
	}
	if errors.Is(err, commandutil.ErrOutputLimit) {
		return Source{}, fmt.Errorf("diff exceeds %d bytes", maxBytes)
	}
	if err != nil {
		return Source{}, fmt.Errorf("collect git diff: %w: %s", err, strings.TrimSpace(string(stderr)))
	}
	rawDiff := string(stdout)
	diff := RedactDiff(rawDiff)
	if int64(len(diff)) > maxBytes {
		return Source{}, fmt.Errorf("scrubbed diff exceeds %d bytes", maxBytes)
	}
	files := diffFiles(rawDiff)
	for _, path := range files {
		if err := validateRelativePath(path); err != nil {
			return Source{}, err
		}
	}
	source := Source{Base: base, Head: head, Diff: diff, RepoRoot: root.Root, ChangedFiles: files}
	if includeUntracked {
		if err := appendUntracked(ctx, root, &source, maxBytes); err != nil {
			return Source{}, err
		}
	}
	hash := sha256.Sum256([]byte(source.Diff))
	source.DiffSHA256 = hex.EncodeToString(hash[:])
	return source, nil
}

// RedactDiff removes known credentials and assignment-style secret values.
func RedactDiff(diff string) string {
	scrubbed := commandutil.Scrub(diff)
	scrubbed = authorizationRE.ReplaceAllString(scrubbed, `${1}[REDACTED]`)
	return diffSecretRE.ReplaceAllString(scrubbed, `${1}[REDACTED]`)
}

func validateRef(ref string) error {
	if ref == "" {
		return nil
	}
	if len(ref) > 256 || strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, "\x00\n\r\t ") {
		return fmt.Errorf("invalid git ref")
	}
	return nil
}

func verifyRef(ctx context.Context, root, ref string) error {
	_, stderr, err := commandutil.RunBounded(ctx, root, []string{"git", "rev-parse", "--verify", ref + "^{commit}"}, commandutil.Limits{Timeout: 10 * time.Second, MaxOutputBytes: 16 * 1024})
	if err != nil {
		return fmt.Errorf("invalid git ref %q: %w: %s", ref, err, strings.TrimSpace(string(stderr)))
	}
	return nil
}

func diffFiles(diff string) []string {
	seen := make(map[string]bool)
	var files []string
	inHeader := false
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "diff --git "))
			fields := strings.Fields(rest)
			if len(fields) == 2 && strings.HasPrefix(fields[1], "b/") && !strings.ContainsAny(rest, "\"") {
				path := strings.TrimPrefix(fields[1], "b/")
				if path != "" && !seen[path] {
					seen[path] = true
					files = append(files, path)
				}
			}
			inHeader = true
			continue
		}
		if strings.HasPrefix(line, "@@") {
			inHeader = false
			continue
		}
		if !inHeader {
			continue
		}
		var path string
		switch {
		case strings.HasPrefix(line, "+++ b/"):
			path = strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "--- a/"):
			path = strings.TrimPrefix(line, "--- a/")
		}
		if path != "" && path != "/dev/null" && !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}
	sort.Strings(files)
	return files
}

func validateRelativePath(path string) error {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" || filepath.IsAbs(path) {
		return fmt.Errorf("git diff contains a path outside the workspace")
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." {
			return fmt.Errorf("git diff contains a path outside the workspace")
		}
	}
	return nil
}
