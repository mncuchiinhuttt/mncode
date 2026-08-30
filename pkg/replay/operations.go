package replay

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
)

// Export copies a redacted trace into a workspace-bound private JSON file.
func (s *Store) Export(ctx context.Context, id, destination string) (string, error) {
	trace, events, err := s.Load(ctx, id)
	if err != nil {
		return "", err
	}
	path := destination
	if strings.TrimSpace(path) == "" {
		path = filepath.Join(s.Dir, id, "export.jsonl")
	}
	if err := rejectDestinationSymlink(s.Workspace, path); err != nil {
		return "", err
	}
	path, err = tools.ResolveWorkspacePath(s.Workspace.Root, path, true)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(struct {
		Trace  Trace   `json:"trace"`
		Events []Event `json:"events"`
	}{trace, events}, "", "  ")
	if err != nil {
		return "", err
	}
	if err := commandutil.WritePrivateJSONExclusive(path, json.RawMessage(data)); err != nil {
		return "", err
	}
	return path, nil
}

// Delete removes a trace only after explicit approval.
func (s *Store) Delete(ctx context.Context, id string, approved bool) error {
	if s == nil {
		return errors.New("replay store is required")
	}
	if !approved {
		return errors.New("deleting replay trace requires explicit approval")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if err := safeTraceID(id); err != nil {
		return err
	}
	relative := filepath.Join(s.Dir, id)
	if err := s.Workspace.RejectSymlinkPath(relative); err != nil {
		return err
	}
	path, err := tools.ResolveWorkspacePath(s.Workspace.Root, relative, false)
	if err != nil {
		return err
	}
	return os.RemoveAll(path)
}

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
