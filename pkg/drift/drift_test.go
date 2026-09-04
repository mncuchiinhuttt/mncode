package drift

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureAndCheckDetectsSignatureChange(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pkg/api/api.go", "package api\n\nfunc Public() string { return \"ok\" }\n")
	sentinel, err := New(root, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := sentinel.Capture(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := sentinel.Save(baseline); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "pkg/api/api.go", "package api\n\nfunc Public() int { return 1 }\n")
	report, err := sentinel.Check(context.Background(), &baseline)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Drifted || report.ExitCode(false) != 1 {
		t.Fatalf("expected blocking drift: %+v", report)
	}
	found := false
	for _, finding := range report.Findings {
		if finding.Code == "signature_changed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("signature change not reported: %+v", report.Findings)
	}
}

func TestForbiddenImportAndCycle(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.24\n")
	writeFile(t, root, "presentation/a.go", "package presentation\n\nimport \"example.com/app/infrastructure\"\n\nfunc A() {}\n")
	writeFile(t, root, "infrastructure/b.go", "package infrastructure\n\nimport \"example.com/app/presentation\"\n\nfunc B() {}\n")
	policy := Policy{DenyCycles: true, Layers: []Layer{{Name: "presentation", Globs: []string{"presentation/**"}}, {Name: "infrastructure", Globs: []string{"infrastructure/**"}}}, ForbiddenImports: map[string][]string{"presentation": {"infrastructure"}}}
	sentinel, err := New(root, policy)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := sentinel.Capture(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	report, err := sentinel.Check(context.Background(), &baseline)
	if err != nil {
		t.Fatal(err)
	}
	// Architecture rules are evaluated on every check, including an unchanged tree.
	codes := map[string]bool{}
	for _, finding := range report.Findings {
		codes[finding.Code] = true
	}
	if !codes["forbidden_import"] || !codes["import_cycle"] {
		t.Fatalf("expected architecture findings: %+v", report.Findings)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
