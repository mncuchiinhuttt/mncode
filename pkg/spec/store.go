package spec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
)

const specDir = ".mncode/spec"

// Store persists versioned contracts inside one canonical workspace.
type Store struct {
	Workspace commandutil.Workspace
	Dir       string
	Limits    commandutil.Limits
}

// New creates a contract store rooted at a canonical workspace.
func New(workspace string) (*Store, error) {
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	return &Store{Workspace: root, Dir: specDir, Limits: commandutil.DefaultLimits()}, nil
}

// NewContract returns a starter contract without writing it.
func (s *Store) NewContract(ctx context.Context, id, title string) (Contract, error) {
	if s == nil {
		return Contract{}, errors.New("spec store is required")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return Contract{}, err
		}
	}
	if strings.TrimSpace(id) == "" {
		id = "feature"
	}
	if !validID(id) {
		return Contract{}, errors.New("invalid spec id")
	}
	contract := Contract{SchemaVersion: 1, ID: id, Title: strings.TrimSpace(title), Description: "Define observable behavior before implementation.", Version: 1, CreatedAt: nowUTC(), Invariants: []Invariant{{ID: "source-safe", Description: "The implementation must not leak credentials or mutate unrelated files.", Kind: "security", Value: json.RawMessage(`true`)}}, Cases: []Case{{ID: "happy-path", Name: "Happy path behavior", Kind: "invariant", Input: json.RawMessage(`{"operator":"non_empty","value":"replace-me"}`), Expected: json.RawMessage(`true`)}}}
	if contract.Title == "" {
		contract.Title = id
	}
	return contract, nil
}

// Save validates and atomically persists a contract.
func (s *Store) Save(ctx context.Context, contract Contract) error {
	if s == nil {
		return errors.New("spec store is required")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if err := Validate(contract); err != nil {
		return err
	}
	if err := s.Workspace.RejectSymlinkPath(filepath.Join(s.Dir, contract.ID+".json")); err != nil {
		return err
	}
	path, err := s.contractPath(contract.ID, true)
	if err != nil {
		return err
	}
	if err := commandutil.WritePrivateJSONExclusive(path, contract); err != nil {
		return fmt.Errorf("spec %q already exists or cannot be written: %w", contract.ID, err)
	}
	return nil
}

// Load reads a contract by safe id or workspace-relative JSON path.
func (s *Store) Load(ctx context.Context, idOrPath string) (Contract, error) {
	if s == nil {
		return Contract{}, errors.New("spec store is required")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return Contract{}, err
		}
	}
	path, err := s.contractPath(idOrPath, false)
	if err != nil {
		return Contract{}, err
	}
	var contract Contract
	if err := commandutil.ReadJSON(path, &contract, 512*1024); err != nil {
		return Contract{}, err
	}
	if err := Validate(contract); err != nil {
		return Contract{}, err
	}
	return contract, nil
}

// List returns contract ids in stable order.
func (s *Store) List(ctx context.Context) ([]string, error) {
	if s == nil {
		return nil, errors.New("spec store is required")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	root, err := tools.ResolveWorkspacePath(s.Workspace.Root, s.Dir, false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			ids = append(ids, strings.TrimSuffix(entry.Name(), ".json"))
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// Export writes a validated contract to a workspace-bound destination.
func (s *Store) Export(ctx context.Context, contract Contract, destination string) (string, error) {
	if s == nil {
		return "", errors.New("spec store is required")
	}
	if err := Validate(contract); err != nil {
		return "", err
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return "", err
		}
	}
	if strings.TrimSpace(destination) == "" {
		destination = filepath.Join(s.Dir, contract.ID+".export.json")
	}
	if err := rejectDestinationSymlink(s.Workspace, destination); err != nil {
		return "", err
	}
	path, err := tools.ResolveWorkspacePath(s.Workspace.Root, destination, true)
	if err != nil {
		return "", err
	}
	if err := commandutil.WritePrivateJSONExclusive(path, contract); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) contractPath(idOrPath string, allowMissing bool) (string, error) {
	value := strings.TrimSpace(idOrPath)
	if value == "" {
		return "", errors.New("spec id or path is required")
	}
	if strings.ContainsAny(value, `/\\`) || strings.HasSuffix(value, ".json") {
		return tools.ResolveWorkspacePath(s.Workspace.Root, value, allowMissing)
	}
	if !validID(value) {
		return "", fmt.Errorf("invalid spec id %q", value)
	}
	return tools.ResolveWorkspacePath(s.Workspace.Root, filepath.Join(s.Dir, value+".json"), allowMissing)
}
func nowUTC() time.Time { return time.Now().UTC() }
func rejectDestinationSymlink(workspace commandutil.Workspace, path string) error {
	if filepath.IsAbs(path) {
		relative, err := workspace.Relative(path)
		if err != nil {
			return err
		}
		path = relative
	}
	return workspace.RejectSymlinkPath(path)
}
