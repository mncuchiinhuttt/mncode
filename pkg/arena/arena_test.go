package arena

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type fakeReviewer struct {
	mu    sync.Mutex
	roles []string
}

func (r *fakeReviewer) Review(_ context.Context, _ Source, role string) ([]Finding, error) {
	r.mu.Lock()
	r.roles = append(r.roles, role)
	r.mu.Unlock()
	return []Finding{{Role: role, Severity: "high", Category: "auth", File: "pkg/auth.go", Line: 3, Evidence: "same evidence", Impact: "risk", Recommendation: "fix"}}, nil
}

func TestReviewMergesDuplicateFindings(t *testing.T) {
	reviewer := &fakeReviewer{}
	arena, err := New(t.TempDir(), reviewer)
	if err != nil {
		t.Fatal(err)
	}
	report, err := arena.Review(context.Background(), Source{RepoRoot: arena.Workspace.Root, Diff: "+ changed"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "block" || len(report.Findings) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	for _, role := range roles {
		if !strings.Contains(report.Findings[0].Role, role) {
			t.Fatalf("merged finding lost reviewer role %q: %q", role, report.Findings[0].Role)
		}
	}
	if len(reviewer.roles) != len(roles) {
		t.Fatalf("expected %d reviewer calls, got %d", len(roles), len(reviewer.roles))
	}
}

func TestCollectSourceRejectsRefInjection(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", root)
	if err := cmd.Run(); err != nil {
		t.Skip("git unavailable")
	}
	if _, err := CollectSource(context.Background(), root, "--bad", "", false, 1024); err == nil {
		t.Fatal("expected ref validation error")
	}
}

func TestCollectSourceIncludesBoundedUntracked(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", root)
	if err := cmd.Run(); err != nil {
		t.Skip("git unavailable")
	}
	path := filepath.Join(root, "new.go")
	if err := os.WriteFile(path, []byte("package new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source, err := CollectSource(context.Background(), root, "", "", true, 64*1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source.Diff, "new.go") {
		t.Fatalf("untracked file missing: %q", source.Diff)
	}
}
func TestRedactDiffRemovesCredentialLikeAssignments(t *testing.T) {
	diff := "+++ b/config.go\n+apiKey = \"AIzaSy1234567890abcdef1234567890\"\n+Authorization: Bearer private-token-abc\n"
	scrubbed := RedactDiff(diff)
	if strings.Contains(scrubbed, "AIzaSy") || strings.Contains(scrubbed, "private-token") {
		t.Fatalf("credential leaked in diff: %s", scrubbed)
	}
}
func TestRedactDiffRemovesJSONCredentialAssignments(t *testing.T) {
	diff := `{"apiKey":"workspace-secret","credentials":"private-value","authorization":"Bearer live-value"}`
	scrubbed := RedactDiff(diff)
	for _, secret := range []string{"workspace-secret", "private-value", "live-value"} {
		if strings.Contains(scrubbed, secret) {
			t.Fatalf("credential leaked in JSON diff: %s", scrubbed)
		}
	}
}
func TestDiffFilesIgnoresPatchContentHeaders(t *testing.T) {
	diff := "diff --git a/safe.go b/safe.go\n--- a/safe.go\n+++ b/safe.go\n@@ -1 +1 @@\n++++ b/../../outside\n"
	files := diffFiles(diff)
	if len(files) != 1 || files[0] != "safe.go" {
		t.Fatalf("unexpected diff files: %#v", files)
	}
}
