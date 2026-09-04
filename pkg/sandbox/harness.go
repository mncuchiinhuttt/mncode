package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
)

// Harness manages isolated fixture copies rooted in one workspace.
type Harness struct {
	Workspace  commandutil.Workspace
	FixtureDir string
	RunDir     string
	Limits     commandutil.Limits
}

// New creates a fixture harness without touching source files.
func New(workspace string) (*Harness, error) {
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	return &Harness{Workspace: root, FixtureDir: filepath.Join(".mncode", "sandbox", "fixtures"), RunDir: filepath.Join(".mncode", "sandbox", "runs"), Limits: commandutil.DefaultLimits()}, nil
}

// Init creates a starter fixture manifest and refuses overwrite.
func (h *Harness) Init(ctx context.Context, id string) (Fixture, error) {
	if err := contextErr(ctx); err != nil {
		return Fixture{}, err
	}
	id, err := safeID(id)
	if err != nil {
		return Fixture{}, err
	}
	relative := filepath.Join(h.FixtureDir, id)
	if err := h.Workspace.RejectSymlinkPath(relative); err != nil {
		return Fixture{}, err
	}
	path, err := h.path(relative)
	if err != nil {
		return Fixture{}, err
	}
	fixture := Fixture{SchemaVersion: 1, ID: id, Name: id, Root: ".", Command: []string{"go", "test", "./..."}, TimeoutSeconds: 30, MaxOutputBytes: h.Limits.MaxOutputBytes}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return Fixture{}, err
	}
	if err := commandutil.WritePrivateJSONExclusive(filepath.Join(path, "fixture.json"), fixture); err != nil {
		return Fixture{}, fmt.Errorf("fixture %q already exists or cannot be initialized: %w", id, err)
	}
	return fixture, nil
}

// List returns valid fixture manifests in stable order.
func (h *Harness) List(ctx context.Context) ([]Fixture, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	root, err := h.path(h.FixtureDir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fixtures := make([]Fixture, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		fixture, loadErr := h.Load(ctx, entry.Name())
		if loadErr != nil {
			return nil, fmt.Errorf("load sandbox fixture %s: %w", entry.Name(), loadErr)
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures, nil
}

// Load reads and validates one fixture manifest.
func (h *Harness) Load(ctx context.Context, id string) (Fixture, error) {
	if err := contextErr(ctx); err != nil {
		return Fixture{}, err
	}
	id, err := safeID(id)
	if err != nil {
		return Fixture{}, err
	}
	if err := h.Workspace.RejectSymlinkPath(filepath.Join(h.FixtureDir, id, "fixture.json")); err != nil {
		return Fixture{}, err
	}
	root, err := h.path(filepath.Join(h.FixtureDir, id))
	if err != nil {
		return Fixture{}, err
	}
	var fixture Fixture
	if err := commandutil.ReadJSON(filepath.Join(root, "fixture.json"), &fixture, 512*1024); err != nil {
		return Fixture{}, err
	}
	if fixture.ID != id || fixture.SchemaVersion != 1 {
		return Fixture{}, fmt.Errorf("malformed fixture %q", id)
	}
	if err := validateFixture(fixture); err != nil {
		return Fixture{}, err
	}
	return fixture, nil
}

func (h *Harness) path(rel string) (string, error) {
	return tools.ResolveWorkspacePath(h.Workspace.Root, rel, true)
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func safeID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "default", nil
	}
	if len(id) > 64 || id == "." || id == ".." || strings.ContainsAny(id, `/\\`) {
		return "", fmt.Errorf("invalid sandbox id")
	}
	for _, r := range id {
		if !(r == '-' || r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return "", fmt.Errorf("invalid sandbox id")
		}
	}
	return id, nil
}

var errFixtureInvalid = errors.New("invalid sandbox fixture")
