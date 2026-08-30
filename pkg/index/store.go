package index

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
	"os"
	"path/filepath"
	"time"
)

var ErrStale = errors.New("code index is stale; run /index rebuild")

const indexDir = ".mncode/index"

type manifest struct {
	SchemaVersion int       `json:"schema_version"`
	WorkspaceRoot string    `json:"workspace_root"`
	WorkspaceID   string    `json:"workspace_id"`
	BuiltAt       time.Time `json:"built_at"`
	Options       Options   `json:"options"`
	DocumentCount int       `json:"document_count"`
	Checksum      string    `json:"documents_sha256"`
}

// Save writes the index manifest and document records atomically.
func (i *Index) Save() error {
	if i == nil {
		return errors.New("index is required")
	}
	root := i.workspace
	if root.Root == "" {
		var err error
		root, err = commandutil.ResolveWorkspace(i.WorkspaceRoot)
		if err != nil {
			return err
		}
	}
	if root.Root != i.WorkspaceRoot || root.Identity != i.WorkspaceID {
		return errors.New("index belongs to another workspace")
	}
	if err := root.RejectSymlinkPath(indexDir); err != nil {
		return err
	}
	dir, err := tools.ResolveWorkspacePath(root.Root, indexDir, true)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	lines := make([]byte, 0, len(i.Documents)*128)
	for _, doc := range i.Documents {
		data, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		if len(data) > 256*1024 {
			return fmt.Errorf("index document %s exceeds 256KB record limit", doc.Path)
		}
		lines = append(lines, []byte(commandutil.Scrub(string(data))+"\n")...)
	}
	checksum := sha256.Sum256(lines)
	if err := writePrivateBytes(filepath.Join(dir, "documents.jsonl"), lines); err != nil {
		return err
	}
	meta := manifest{SchemaVersion: schemaVersion, WorkspaceRoot: root.Root, WorkspaceID: root.Identity, BuiltAt: i.BuiltAt, Options: i.Options, DocumentCount: len(i.Documents), Checksum: hex.EncodeToString(checksum[:])}
	if err := commandutil.WritePrivateJSON(filepath.Join(dir, "manifest.json"), meta); err != nil {
		return err
	}
	i.workspace = root
	return nil
}

// Open loads and verifies the current workspace index and source hashes.
func Open(workspace string) (*Index, error) {
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	dir, err := tools.ResolveWorkspacePath(root.Root, indexDir, false)
	if err != nil {
		return nil, err
	}
	var meta manifest
	if err := commandutil.ReadJSON(filepath.Join(dir, "manifest.json"), &meta, 512*1024); err != nil {
		return nil, err
	}
	if meta.SchemaVersion != schemaVersion || meta.WorkspaceID != root.Identity || meta.WorkspaceRoot != root.Root {
		return nil, errors.New("index belongs to another workspace or schema")
	}
	if meta.DocumentCount < 0 || meta.DocumentCount > commandutil.DefaultLimits().MaxFiles {
		return nil, errors.New("index document count is invalid")
	}
	if meta.Options.MaxFiles < 0 || meta.Options.MaxFiles > commandutil.DefaultLimits().MaxFiles ||
		meta.Options.MaxFileBytes < 0 || meta.Options.MaxFileBytes > commandutil.DefaultLimits().MaxFileBytes {
		return nil, errors.New("index options are invalid")
	}
	documentsPath := filepath.Join(dir, "documents.jsonl")
	info, err := os.Stat(documentsPath)
	if err != nil {
		return nil, err
	}
	if info.Size() > 32*1024*1024 {
		return nil, errors.New("index document file is too large")
	}
	file, err := os.Open(documentsPath)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, 32*1024*1024+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(data) > 32*1024*1024 {
		return nil, errors.New("index document file is too large")
	}
	checksum := sha256.Sum256(data)
	if hex.EncodeToString(checksum[:]) != meta.Checksum {
		return nil, errors.New("index checksum mismatch")
	}
	idx := &Index{SchemaVersion: meta.SchemaVersion, WorkspaceRoot: root.Root, WorkspaceID: root.Identity, BuiltAt: meta.BuiltAt, Options: meta.Options, Documents: make([]Document, 0, meta.DocumentCount), Terms: make(map[string][]Posting), workspace: root}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 2*1024*1024)
	seenIDs := make(map[string]bool, meta.DocumentCount)
	for scanner.Scan() {
		var doc Document
		if err := json.Unmarshal(scanner.Bytes(), &doc); err != nil {
			return nil, err
		}
		if doc.Path != doc.ID || doc.ID == "" || seenIDs[doc.ID] {
			return nil, errors.New("index document identity is invalid or duplicated")
		}
		seenIDs[doc.ID] = true
		idx.Documents = append(idx.Documents, doc)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(idx.Documents) != meta.DocumentCount {
		return nil, errors.New("index document count mismatch")
	}
	for _, doc := range idx.Documents {
		if staleDocument(root.Root, doc) {
			return nil, fmt.Errorf("%w: %s", ErrStale, doc.Path)
		}
	}
	fresh, err := Build(context.Background(), root.Root, meta.Options)
	if err != nil {
		return nil, err
	}
	if len(fresh.Documents) != len(idx.Documents) {
		return nil, ErrStale
	}
	for documentIndex := range idx.Documents {
		if fresh.Documents[documentIndex].Path != idx.Documents[documentIndex].Path ||
			fresh.Documents[documentIndex].SHA256 != idx.Documents[documentIndex].SHA256 {
			return nil, fmt.Errorf("%w: %s", ErrStale, idx.Documents[documentIndex].Path)
		}
	}
	idx.rebuildPostings()
	return idx, nil
}

// Clear removes the persisted index only after explicit approval.
func Clear(workspace string, approved bool) error {
	if !approved {
		return errors.New("clearing code index requires explicit approval")
	}
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return err
	}
	if err := root.RejectSymlinkPath(indexDir); err != nil {
		return err
	}
	dir, err := tools.ResolveWorkspacePath(root.Root, indexDir, false)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}
