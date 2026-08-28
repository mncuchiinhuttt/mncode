// Package agent — git worktree isolation for subagents. When worktree_base
// is "main" or "fresh", a spawned subagent runs against a dedicated git
// worktree + branch instead of sharing the parent's live working directory,
// so it can never race the user's own uncommitted edits or another
// subagent's concurrent file changes. "current" (the default) keeps the
// pre-existing behavior of operating directly on the shared workspace.
package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// SubagentWorktree describes an isolated git worktree created for one
// subagent run, plus how to tear it down afterward.
type SubagentWorktree struct {
	// Path is the isolated working directory the subagent should use in
	// place of the parent's WorkspaceDir.
	Path string
	// Branch is the throwaway branch checked out in Path. It is left in the
	// repo after cleanup so the user can inspect or merge the subagent's
	// changes with `git log`/`git diff`/`git merge` — only the worktree
	// checkout is removed, never the branch or its commits.
	Branch string
	// BaseRef is the ref the branch was created from (e.g. "main", "HEAD").
	BaseRef string
}

// isGitRepository reports whether dir is inside a git working tree.
func isGitRepository(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// refExists reports whether ref resolves to a commit in dir's repo.
func refExists(dir, ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", ref)
	cmd.Dir = dir
	return cmd.Run() == nil
}

// resolveBaseRef maps the worktree_base setting to an actual git ref,
// falling back sensibly when the preferred branch name doesn't exist.
func resolveBaseRef(workspaceDir, worktreeBase string) (string, error) {
	switch worktreeBase {
	case "main":
		for _, candidate := range []string{"main", "master"} {
			if refExists(workspaceDir, candidate) {
				return candidate, nil
			}
		}
		return "", fmt.Errorf("no 'main' or 'master' branch found")
	case "fresh":
		// HEAD at the last commit, deliberately excluding any of the user's
		// uncommitted working-directory changes.
		if !refExists(workspaceDir, "HEAD") {
			return "", fmt.Errorf("repository has no commits yet")
		}
		return "HEAD", nil
	default:
		return "", fmt.Errorf("unsupported worktree base: %s", worktreeBase)
	}
}

var subagentBranchSeq uint64

func newSubagentBranch(subID string) string {
	sequence := atomic.AddUint64(&subagentBranchSeq, 1)
	return fmt.Sprintf(
		"mncode/subagent-%s-%d-%d",
		sanitizeBranchComponent(subID),
		time.Now().UnixNano(),
		sequence,
	)
}

// CreateSubagentWorktree creates an isolated git worktree checked out from
// worktreeBase ("main" or "fresh") on a new throwaway branch, for a subagent
// identified by subID to run in. Returns (nil, nil) when isolation isn't
// applicable — worktreeBase is "current", or workspaceDir isn't a git repo
// — so callers can fall back to sharing the parent's workspace directly.
func CreateSubagentWorktree(workspaceDir, subID, worktreeBase string) (*SubagentWorktree, error) {
	if worktreeBase == "" || worktreeBase == "current" {
		return nil, nil
	}
	if strings.TrimSpace(workspaceDir) == "" || !isGitRepository(workspaceDir) {
		return nil, nil
	}

	baseRef, err := resolveBaseRef(workspaceDir, worktreeBase)
	if err != nil {
		return nil, fmt.Errorf("worktree isolation unavailable: %w", err)
	}

	branch := newSubagentBranch(subID)

	worktreeRoot := filepath.Join(os.TempDir(), "mncode-worktrees")
	if err := os.MkdirAll(worktreeRoot, 0o700); err != nil {
		return nil, fmt.Errorf("prepare worktree directory: %w", err)
	}
	if err := os.Chmod(worktreeRoot, 0o700); err != nil {
		return nil, fmt.Errorf("secure worktree directory: %w", err)
	}
	worktreePath := filepath.Join(worktreeRoot, branch)
	// Git refuses to create a worktree at an already-existing non-empty
	// directory; a stale one from a crashed prior run shouldn't block us.
	_ = os.RemoveAll(worktreePath)

	cmd := exec.Command("git", "worktree", "add", "-b", branch, worktreePath, baseRef)
	cmd.Dir = workspaceDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git worktree add failed: %s", strings.TrimSpace(stderr.String()))
	}

	return &SubagentWorktree{Path: worktreePath, Branch: branch, BaseRef: baseRef}, nil
}

// Cleanup preserves any uncommitted changes on the isolated branch, then
// removes only the checkout. If preserving changes fails, the checkout is
// deliberately left in place so the user can recover the work manually.
func (w *SubagentWorktree) Cleanup(workspaceDir string) error {
	if w == nil {
		return nil
	}
	if err := w.preserveChanges(); err != nil {
		return fmt.Errorf("preserve subagent worktree changes: %w", err)
	}

	cmd := exec.Command("git", "worktree", "remove", "--force", w.Path)
	cmd.Dir = workspaceDir
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Worktree metadata may already be inconsistent (e.g. the directory
	// was manually deleted). Fall back to pruning + best-effort removal
	// so we never leak an entry in `git worktree list`.
	if err := os.RemoveAll(w.Path); err != nil {
		return fmt.Errorf("remove worktree checkout: %w", err)
	}
	pruneCmd := exec.Command("git", "worktree", "prune")
	pruneCmd.Dir = workspaceDir
	if err := pruneCmd.Run(); err != nil {
		return fmt.Errorf("prune stale worktree metadata: %w", err)
	}
	return nil
}

func (w *SubagentWorktree) preserveChanges() error {
	statusCmd := exec.Command("git", "status", "--porcelain", "--untracked-files=all")
	statusCmd.Dir = w.Path
	status, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("inspect worktree status: %w", err)
	}
	if strings.TrimSpace(string(status)) == "" {
		return nil
	}
	for _, path := range worktreeStatusPaths(status) {
		if isSensitiveWorktreePath(path) {
			return fmt.Errorf("refusing to auto-commit sensitive path %q; recover the isolated worktree manually", path)
		}
	}

	addCmd := exec.Command("git", "add", "--all")
	addCmd.Dir = w.Path
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("stage changes: %w: %s", err, strings.TrimSpace(string(output)))
	}

	commitCmd := exec.Command(
		"git", "-c", "user.name=mncode subagent", "-c",
		"user.email=mncode-subagent@localhost", "commit", "--no-gpg-sign",
		"-m", "chore(mncode): preserve subagent changes",
	)
	commitCmd.Dir = w.Path
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("commit preserved changes: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func worktreeStatusPaths(raw []byte) []string {
	var paths []string
	for _, line := range strings.Split(string(raw), "\n") {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if path == "" {
			continue
		}
		if parts := strings.SplitN(path, " -> ", 2); len(parts) == 2 {
			paths = append(paths, strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

func isSensitiveWorktreePath(path string) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(path)))
	if base == ".env" || (strings.HasPrefix(base, ".env.") && !strings.Contains(base, "example") && !strings.Contains(base, "sample")) {
		return true
	}
	for _, suffix := range []string{".pem", ".key", ".p12", ".pfx", ".jks", ".keystore"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	for _, marker := range []string{"credential", "secret", "private-key", "id_rsa", "id_ed25519"} {
		if strings.Contains(base, marker) {
			return true
		}
	}
	return false
}

// sanitizeBranchComponent keeps only characters safe for a git ref segment.
func sanitizeBranchComponent(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	result := b.String()
	if result == "" {
		return "subagent"
	}
	return result
}
