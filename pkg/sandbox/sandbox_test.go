package sandbox

import (
	"context"
	"encoding/json"
	"mncode/pkg/commandutil"
	"os"
	"path/filepath"
	"testing"
)

func TestInitRunAndViewPreserveFixture(t *testing.T) {
	root := t.TempDir()
	h, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := h.Init(context.Background(), "echo-case")
	if err != nil {
		t.Fatal(err)
	}
	fixture.Command = []string{"printf", "sandbox-ok"}
	manifest := filepath.Join(root, ".mncode", "sandbox", "fixtures", fixture.ID, "fixture.json")
	data, _ := json.Marshal(fixture)
	if err := os.WriteFile(manifest, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Init(context.Background(), fixture.ID); err == nil {
		t.Fatal("expected duplicate init rejection")
	}
	result, err := h.Run(context.Background(), RunRequest{FixtureID: fixture.ID, Keep: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || result.Stdout != "sandbox-ok" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := h.View(context.Background(), result.ID); err != nil {
		t.Fatal(err)
	}
	fixtureCopy := filepath.Join(root, ".mncode", "sandbox", "fixtures", fixture.ID, "fixture.json")
	if _, err := os.Stat(fixtureCopy); err != nil {
		t.Fatal(err)
	}
	if err := h.Clean(context.Background(), result.ID, false); err == nil {
		t.Fatal("expected approval requirement")
	}
	if err := h.Clean(context.Background(), result.ID, true); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsShellSyntaxAndSecretEnvironment(t *testing.T) {
	fixture := Fixture{SchemaVersion: 1, ID: "x", Root: ".", Command: []string{"printf", "ok;touch bad"}}
	if err := validateFixture(fixture); err == nil {
		t.Fatal("expected shell syntax rejection")
	}
	fixture.Command = []string{"printf", "ok"}
	fixture.Env = map[string]string{"MNCODE_API_KEY": "secret"}
	if err := validateFixture(fixture); err == nil {
		t.Fatal("expected secret environment rejection")
	}
}
func TestCommandCannotEmbedSourceWorkspacePath(t *testing.T) {
	root := t.TempDir()
	h, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = h.RunCommand(context.Background(), []string{"python3", "-c", "open('" + root + "/victim', 'w').write('x')"}, nil, commandutil.DefaultLimits())
	if err == nil {
		t.Fatal("expected source path rejection")
	}
}
