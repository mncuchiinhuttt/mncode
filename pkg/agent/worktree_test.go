package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")
	return dir
}

func TestSubagentBranchNameIsUnique(t *testing.T) {
	first := newSubagentBranch("same-agent")
	second := newSubagentBranch("same-agent")
	if first == second {
		t.Fatalf("expected unique branch names, got %q twice", first)
	}
}

func TestCleanupPreservesUncommittedChanges(t *testing.T) {
	dir := newTestRepo(t)
	wt, err := CreateSubagentWorktree(dir, "sub-uncommitted", "fresh")
	if err != nil {
		t.Fatalf("CreateSubagentWorktree failed: %v", err)
	}
	if wt == nil {
		t.Fatal("expected a worktree")
	}

	kept := filepath.Join(wt.Path, "kept.txt")
	if err := os.WriteFile(kept, []byte("preserve me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := wt.Cleanup(dir); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	if _, statErr := os.Stat(wt.Path); !os.IsNotExist(statErr) {
		t.Fatalf("expected worktree path to be removed, stat err: %v", statErr)
	}

	content := runGit(t, dir, "show", wt.Branch+":kept.txt")
	if content != "preserve me\n" {
		t.Fatalf("preserved branch content = %q, want %q", content, "preserve me\\n")
	}
}

func TestCleanupRefusesSensitiveChanges(t *testing.T) {
	dir := newTestRepo(t)
	wt, err := CreateSubagentWorktree(dir, "sub-sensitive", "fresh")
	if err != nil {
		t.Fatalf("CreateSubagentWorktree failed: %v", err)
	}
	if wt == nil {
		t.Fatal("expected a worktree")
	}

	secretFile := filepath.Join(wt.Path, ".env")
	if err := os.WriteFile(secretFile, []byte("API_KEY=do-not-commit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := wt.Cleanup(dir); err == nil {
		t.Fatal("expected Cleanup to refuse a sensitive path")
	}
	if _, statErr := os.Stat(secretFile); statErr != nil {
		t.Fatalf("sensitive worktree should remain recoverable: %v", statErr)
	}
}

// TestCreateSubagentWorktree_Current verifies "current" (default) performs
// no isolation at all — same behavior as before this feature existed.
func TestCreateSubagentWorktree_Current(t *testing.T) {
	dir := newTestRepo(t)
	wt, err := CreateSubagentWorktree(dir, "sub1", "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt != nil {
		t.Fatalf("expected nil worktree for 'current' base, got %+v", wt)
	}
}

// TestCreateSubagentWorktree_NonGitDir verifies isolation is skipped (not
// an error) when the workspace isn't a git repository at all.
func TestCreateSubagentWorktree_NonGitDir(t *testing.T) {
	dir := t.TempDir()
	wt, err := CreateSubagentWorktree(dir, "sub1", "fresh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt != nil {
		t.Fatalf("expected nil worktree for non-git dir, got %+v", wt)
	}
}

// TestCreateSubagentWorktree_Fresh exercises the real path: creates an
// actual git worktree on a real throwaway branch from HEAD, verifies it is
// isolated from the parent repo's working directory, writes a file inside
// it, and confirms Cleanup removes the worktree checkout while preserving
// the branch and its commits in the main repo.
func TestCreateSubagentWorktree_Fresh(t *testing.T) {
	dir := newTestRepo(t)

	wt, err := CreateSubagentWorktree(dir, "sub-fresh", "fresh")
	if err != nil {
		t.Fatalf("CreateSubagentWorktree failed: %v", err)
	}
	if wt == nil {
		t.Fatalf("expected a worktree, got nil")
	}
	if _, statErr := os.Stat(wt.Path); statErr != nil {
		t.Fatalf("worktree path does not exist: %v", statErr)
	}
	if wt.Path == dir {
		t.Fatalf("worktree path must differ from parent workspace dir")
	}

	// Write a file inside the isolated worktree and commit it there.
	isolatedFile := filepath.Join(wt.Path, "isolated.txt")
	if err := os.WriteFile(isolatedFile, []byte("from subagent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, wt.Path, "add", ".")
	runGit(t, wt.Path, "commit", "-m", "subagent change")

	// The parent repo's working directory must be completely unaffected.
	if _, statErr := os.Stat(filepath.Join(dir, "isolated.txt")); statErr == nil {
		t.Fatalf("isolated.txt leaked into the parent workspace")
	}

	branchListOut := runGit(t, dir, "branch", "--list", wt.Branch)
	if !strings.Contains(branchListOut, wt.Branch) {
		t.Fatalf("expected branch %s to exist in parent repo, got: %s", wt.Branch, branchListOut)
	}

	wt.Cleanup(dir)

	if _, statErr := os.Stat(wt.Path); !os.IsNotExist(statErr) {
		t.Fatalf("expected worktree path to be removed after Cleanup, stat err: %v", statErr)
	}

	// Branch + commit must survive cleanup so the user can still find/merge it.
	branchListOut = runGit(t, dir, "branch", "--list", wt.Branch)
	if !strings.Contains(branchListOut, wt.Branch) {
		t.Fatalf("expected branch %s to survive Cleanup, got: %s", wt.Branch, branchListOut)
	}
	logOut := runGit(t, dir, "log", wt.Branch, "--oneline")
	if !strings.Contains(logOut, "subagent change") {
		t.Fatalf("expected subagent commit to survive on branch, got: %s", logOut)
	}
}

// TestCreateSubagentWorktree_MainMissing verifies a clear error (not a
// silent no-op) when worktree_base=main is requested but no main/master
// branch exists.
func TestCreateSubagentWorktree_MainMissing(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "trunk")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "c1")

	_, err := CreateSubagentWorktree(dir, "sub2", "main")
	if err == nil {
		t.Fatalf("expected an error when no main/master branch exists")
	}
}
