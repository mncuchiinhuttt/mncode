package replay

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
)

// List returns completed and incomplete trace manifests in stable order.
func (s *Store) List(ctx context.Context) ([]Trace, error) {
	if s == nil {
		return nil, errors.New("replay store is required")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	root, err := tools.ResolveWorkspacePath(s.Workspace.Root, s.Dir, false)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	traces := make([]Trace, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if err := safeTraceID(entry.Name()); err != nil {
			return nil, fmt.Errorf("invalid replay directory %q", entry.Name())
		}
		if err := s.Workspace.RejectSymlinkPath(filepath.Join(s.Dir, entry.Name(), "manifest.json")); err != nil {
			return nil, err
		}
		var trace Trace
		if err := commandutil.ReadJSON(filepath.Join(root, entry.Name(), "manifest.json"), &trace, 512*1024); err != nil {
			return nil, fmt.Errorf("load replay %s: %w", entry.Name(), err)
		}
		if trace.ID != entry.Name() || trace.WorkspaceID != s.Workspace.Identity || trace.WorkspaceRoot != s.Workspace.Root {
			return nil, fmt.Errorf("replay %s belongs to another workspace", entry.Name())
		}
		traces = append(traces, trace)
	}
	sort.Slice(traces, func(i, j int) bool { return traces[i].StartedAt.After(traces[j].StartedAt) })
	return traces, nil
}

// Load reads a trace manifest and ordered event stream.
func (s *Store) Load(ctx context.Context, id string) (Trace, []Event, error) {
	if s == nil {
		return Trace{}, nil, errors.New("replay store is required")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return Trace{}, nil, err
		}
	}
	maxBytes := s.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 16 * 1024 * 1024
	}
	maxEvents := s.MaxEvents
	if maxEvents <= 0 {
		maxEvents = 2000
	}
	if err := safeTraceID(id); err != nil {
		return Trace{}, nil, err
	}
	if err := s.Workspace.RejectSymlinkPath(filepath.Join(s.Dir, id, "manifest.json")); err != nil {
		return Trace{}, nil, err
	}
	if err := s.Workspace.RejectSymlinkPath(filepath.Join(s.Dir, id, "trace.jsonl")); err != nil {
		return Trace{}, nil, err
	}
	root, err := tools.ResolveWorkspacePath(s.Workspace.Root, filepath.Join(s.Dir, id), false)
	if err != nil {
		return Trace{}, nil, err
	}
	var trace Trace
	if err := commandutil.ReadJSON(filepath.Join(root, "manifest.json"), &trace, 512*1024); err != nil {
		return Trace{}, nil, err
	}
	if trace.SchemaVersion != 1 || trace.ID != id || trace.WorkspaceID != s.Workspace.Identity || trace.WorkspaceRoot != s.Workspace.Root {
		return Trace{}, nil, errors.New("trace identity mismatch")
	}
	if trace.Events < 0 || trace.Events > maxEvents {
		return Trace{}, nil, errors.New("trace event count is invalid")
	}
	if trace.Complete && trace.Checksum == "" {
		return Trace{}, nil, errors.New("completed trace is missing checksum")
	}
	tracePath := filepath.Join(root, "trace.jsonl")
	info, err := os.Stat(tracePath)
	if err != nil {
		return Trace{}, nil, err
	}
	if info.Size() > maxBytes {
		return Trace{}, nil, errors.New("trace exceeds configured size limit")
	}
	file, err := os.Open(tracePath)
	if err != nil {
		return Trace{}, nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return Trace{}, nil, readErr
	}
	if closeErr != nil {
		return Trace{}, nil, closeErr
	}
	if int64(len(data)) > maxBytes {
		return Trace{}, nil, errors.New("trace exceeds configured size limit")
	}
	digest := sha256.Sum256(data)
	if trace.Checksum != "" && trace.Checksum != hex.EncodeToString(digest[:]) {
		return Trace{}, nil, errors.New("trace checksum mismatch")
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024), 512*1024)
	events := make([]Event, 0, trace.Events)
	expectedSeq := int64(1)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return Trace{}, nil, err
		}
		if event.Seq != expectedSeq || event.Kind == "" {
			return Trace{}, nil, errors.New("trace sequence is invalid")
		}
		expectedSeq++
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return Trace{}, nil, err
	}
	if len(events) > maxEvents {
		return Trace{}, nil, errors.New("trace event count is invalid")
	}
	if trace.Complete && trace.Events != len(events) {
		return Trace{}, nil, errors.New("trace event count mismatch")
	}
	if !trace.Complete {
		trace.Events = len(events)
	}
	return trace, events, nil
}
