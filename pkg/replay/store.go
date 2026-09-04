package replay

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
)

const replayDir = ".mncode/replay"

// NewStore creates a private trace store rooted at a canonical workspace.
func NewStore(workspace string) (*Store, error) {
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	return &Store{Workspace: root, Dir: replayDir, MaxEvents: 2000, MaxBytes: 16 * 1024 * 1024}, nil
}

// NewRecorder is the constructor form of Store.Start.
func NewRecorder(store *Store, sessionID string, meta Trace) (*Recorder, error) {
	if store == nil {
		return nil, errors.New("replay store is required")
	}
	return store.Start(context.Background(), sessionID, meta)
}

// Start creates and opens a new append-only recorder.
func (s *Store) Start(ctx context.Context, sessionID string, meta Trace) (*Recorder, error) {
	if s == nil {
		return nil, errors.New("replay store is required")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("session id is required")
	}
	id := commandutil.NewID("trace")
	meta.SchemaVersion, meta.ID, meta.SessionID, meta.WorkspaceRoot, meta.WorkspaceID = 1, id, sessionID, s.Workspace.Root, s.Workspace.Identity
	meta.StartedAt = time.Now().UTC()
	meta.Complete = false
	relative := filepath.Join(s.Dir, id)
	if err := s.Workspace.RejectSymlinkPath(relative); err != nil {
		return nil, err
	}
	dir, err := tools.ResolveWorkspacePath(s.Workspace.Root, filepath.Join(s.Dir, id), true)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(dir, "trace.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	recorder := &Recorder{store: s, trace: meta, file: file, nextSeq: 1}
	if err := recorder.Record(KindSessionStart, 0, map[string]string{"session_id": sessionID}); err != nil {
		_ = file.Close()
		return nil, err
	}
	return recorder, nil
}
