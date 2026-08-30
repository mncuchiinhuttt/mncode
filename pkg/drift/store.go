package drift

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
)

const baselineFile = ".mncode/drift/baseline.json"

// Save persists a baseline atomically inside the workspace.
func (s *Sentinel) Save(baseline Baseline) error {
	if s == nil {
		return errors.New("drift sentinel is required")
	}
	if baseline.WorkspaceID != s.Workspace.Identity || baseline.WorkspaceRoot != s.Workspace.Root {
		return errors.New("baseline belongs to another workspace")
	}
	if data, err := json.Marshal(baseline); err != nil || len(data) > 4*1024*1024 {
		return fmt.Errorf("drift baseline exceeds 4MB persistence limit")
	}
	if err := s.Workspace.RejectSymlinkPath(baselineFile); err != nil {
		return err
	}
	path, err := tools.ResolveWorkspacePath(s.Workspace.Root, baselineFile, true)
	if err != nil {
		return err
	}
	return commandutil.WritePrivateJSON(path, baseline)
}

// Load reads the current baseline and verifies its workspace identity.
func (s *Sentinel) Load() (*Baseline, error) {
	if s == nil {
		return nil, errors.New("drift sentinel is required")
	}
	if err := s.Workspace.RejectSymlinkPath(baselineFile); err != nil {
		return nil, err
	}
	path, err := tools.ResolveWorkspacePath(s.Workspace.Root, baselineFile, false)
	if err != nil {
		return nil, err
	}
	var baseline Baseline
	if err := commandutil.ReadJSON(path, &baseline, 4*1024*1024); err != nil {
		return nil, err
	}
	if baseline.SchemaVersion != 1 || strings.TrimSpace(baseline.ID) == "" {
		return nil, fmt.Errorf("unsupported or malformed drift baseline")
	}
	if baseline.WorkspaceID != s.Workspace.Identity || baseline.WorkspaceRoot != s.Workspace.Root {
		return nil, errors.New("baseline belongs to another workspace")
	}
	return &baseline, nil
}

// Delete removes the baseline only after explicit approval.
func (s *Sentinel) Delete(ctx context.Context, approved bool) error {
	if s == nil {
		return errors.New("drift sentinel is required")
	}
	if !approved {
		return errors.New("deleting drift baseline requires explicit approval")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if err := s.Workspace.RejectSymlinkPath(baselineFile); err != nil {
		return err
	}
	path, err := tools.ResolveWorkspacePath(s.Workspace.Root, baselineFile, false)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return nil
}

// BaselinePath returns the workspace-relative baseline location.
func BaselinePath() string { return filepath.ToSlash(baselineFile) }
