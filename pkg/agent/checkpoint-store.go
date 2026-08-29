package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ErrRollbackApprovalRequired is returned before any filesystem or history
// mutation when a caller has not explicitly approved a rollback.
var ErrRollbackApprovalRequired = errors.New("rollback requires explicit approval")

// CheckpointFile is the immutable before/after record for one run-owned path.
// Blob paths are relative to the checkpoint directory and are never followed
// through symlinks during restore.
type CheckpointFile struct {
	Path        string `json:"path"`
	BeforeHash  string `json:"before_hash,omitempty"`
	AfterHash   string `json:"after_hash,omitempty"`
	BeforeBlob  string `json:"before_blob,omitempty"`
	AfterBlob   string `json:"after_blob,omitempty"`
	BeforeMode  uint32 `json:"before_mode,omitempty"`
	AfterMode   uint32 `json:"after_mode,omitempty"`
	Owned       bool   `json:"owned"`
	BeforeExist bool   `json:"before_exists"`
	AfterExist  bool   `json:"after_exists"`
}

// Checkpoint is the adapter record described by persistence-contract.md. The
// session/run identity and revision make a checkpoint unusable from another
// workspace or session; Phase 2 may persist this record in RunStore without
// changing its ownership semantics.
type Checkpoint struct {
	SchemaVersion int              `json:"schema_version"`
	ID            string           `json:"id"`
	SessionID     string           `json:"session_id"`
	RunID         string           `json:"run_id"`
	WorkspaceDir  string           `json:"workspace_dir"`
	StoreRevision string           `json:"store_revision"`
	TurnIndex     int              `json:"turn_index"`
	Timestamp     time.Time        `json:"timestamp"`
	Summary       string           `json:"summary"`
	PatchFile     string           `json:"patch_file,omitempty"`
	Manifest      []CheckpointFile `json:"manifest"`
	Completed     bool             `json:"completed"`
}

// RollbackPlan describes what an approved rollback would do. It is also useful
// to present a confirmation prompt without changing the workspace.
type RollbackPlan struct {
	CheckpointID string
	Restore      []string
	Remove       []string
	Conflicts    []string
	Skipped      []string
}

// CreateTurnCheckpoint records a workspace baseline. It deliberately does not
// infer ownership from git status: callers must complete the checkpoint with
// the paths changed by their run via FinalizeTurnCheckpoint. This avoids
// treating unrelated user edits as agent-owned changes.
func (s *Session) CreateTurnCheckpoint(turnIndex int, summary string) (*Checkpoint, error) {
	// The CLI historically used mncode-main for its process-wide session. A
	// checkpoint must instead be tied to the durable session that owns it.
	sessionID := ensureSessionIdentity(s)
	workspace, err := checkpointWorkspace(s.WorkspaceDir)
	if err != nil {
		return nil, err
	}
	checkpointDir := filepath.Join(workspace, ".mncode", "checkpoints")
	if err := os.MkdirAll(filepath.Join(checkpointDir, "blobs"), 0o700); err != nil {
		return nil, fmt.Errorf("create checkpoint directory: %w", err)
	}
	_ = os.Chmod(filepath.Join(workspace, ".mncode"), 0o700)
	_ = os.Chmod(checkpointDir, 0o700)

	before, err := captureWorkspace(workspace, checkpointDir)
	if err != nil {
		return nil, fmt.Errorf("capture checkpoint baseline: %w", err)
	}
	id := fmt.Sprintf("ckpt-%d-%d", turnIndex, time.Now().UnixNano())
	cp := &Checkpoint{
		SchemaVersion: 1,
		ID:            id,
		SessionID:     sessionID,
		RunID:         id,
		WorkspaceDir:  workspace,
		StoreRevision: fmt.Sprintf("session:%s:turn:%d", sessionID, turnIndex),
		TurnIndex:     turnIndex,
		Timestamp:     time.Now().UTC(),
		Summary:       summary,
		Manifest:      make([]CheckpointFile, 0, len(before)),
	}
	for path, snap := range before {
		blob, err := saveBlob(checkpointDir, snap)
		if err != nil {
			return nil, fmt.Errorf("save baseline for %s: %w", path, err)
		}
		cp.Manifest = append(cp.Manifest, CheckpointFile{
			Path: path, BeforeHash: snap.Hash, BeforeBlob: blob,
			BeforeMode: snap.Mode, BeforeExist: true,
		})
	}
	sort.Slice(cp.Manifest, func(i, j int) bool { return cp.Manifest[i].Path < cp.Manifest[j].Path })
	if err := persistCheckpoint(cp, checkpointDir); err != nil {
		return nil, err
	}
	return cp, nil
}

// FinalizeTurnCheckpoint records the after-state for explicit run-owned paths.
// A path may be newly created or deleted. Omitting paths is rejected rather
// than silently claiming the entire workspace for the run.
func (s *Session) FinalizeTurnCheckpoint(cp *Checkpoint, ownedPaths ...string) error {
	if cp == nil {
		return errors.New("cannot finalize a nil checkpoint")
	}
	if err := validateCheckpointIdentity(s, cp); err != nil {
		return err
	}
	if len(ownedPaths) == 0 {
		return errors.New("checkpoint completion requires at least one run-owned path")
	}
	workspace, err := checkpointWorkspace(s.WorkspaceDir)
	if err != nil {
		return err
	}
	normalized := make([]string, 0, len(ownedPaths))
	seen := make(map[string]struct{}, len(ownedPaths))
	for _, raw := range ownedPaths {
		path, err := cleanCheckpointPath(raw)
		if err != nil {
			return err
		}
		if strings.HasPrefix(path, ".mncode/") || path == ".git" || strings.HasPrefix(path, ".git/") {
			return fmt.Errorf("checkpoint path %q is reserved", raw)
		}
		if _, ok := seen[path]; !ok {
			seen[path] = struct{}{}
			normalized = append(normalized, path)
		}
	}
	if len(normalized) == 0 {
		return errors.New("checkpoint completion requires at least one run-owned path")
	}
	checkpointDir := filepath.Join(workspace, ".mncode", "checkpoints")
	current, err := captureWorkspace(workspace, checkpointDir)
	if err != nil {
		return fmt.Errorf("capture checkpoint result: %w", err)
	}
	// Store indexes rather than pointers: appending a new manifest entry may
	// reallocate the slice, invalidating pointers into the previous backing
	// array. Sorting is performed only after all indexes have been consumed.
	byPath := make(map[string]int, len(cp.Manifest))
	for i := range cp.Manifest {
		byPath[cp.Manifest[i].Path] = i
	}
	for _, path := range normalized {
		index, ok := byPath[path]
		if !ok {
			cp.Manifest = append(cp.Manifest, CheckpointFile{Path: path})
			index = len(cp.Manifest) - 1
			byPath[path] = index
		}
		entry := &cp.Manifest[index]
		entry.Owned = true
		entry.AfterBlob = ""
		entry.AfterExist = false
		entry.AfterHash = ""
		entry.AfterMode = 0
		if snap, ok := current[path]; ok {
			blob, err := saveBlob(checkpointDir, snap)
			if err != nil {
				return fmt.Errorf("save result for %s: %w", path, err)
			}
			entry.AfterHash, entry.AfterBlob = snap.Hash, blob
			entry.AfterMode, entry.AfterExist = snap.Mode, true
		}
	}
	sort.Slice(cp.Manifest, func(i, j int) bool { return cp.Manifest[i].Path < cp.Manifest[j].Path })
	cp.Completed = true
	return persistCheckpoint(cp, checkpointDir)
}

// CompleteTurnCheckpoint is an explicit alias for callers that use the
// persistence contract's terminology.
func (s *Session) CompleteTurnCheckpoint(cp *Checkpoint, ownedPaths ...string) error {
	return s.FinalizeTurnCheckpoint(cp, ownedPaths...)
}

// RollbackLastTurn requires an explicit true approval. The variadic form keeps
// source compatibility with the old no-argument caller while making that path
// fail closed; no-argument and false calls have no side effects.
func (s *Session) RollbackLastTurn(approval ...bool) (string, error) {
	if len(approval) != 1 || !approval[0] {
		return "", ErrRollbackApprovalRequired
	}
	if len(s.History) == 0 {
		return "", fmt.Errorf("no conversation history to undo")
	}
	checkpoints, err := s.ListCheckpoints()
	if err != nil {
		return "", err
	}
	for _, cp := range checkpoints {
		if cp.Completed && cp.SessionID == s.ID && sameWorkspace(cp.WorkspaceDir, s.WorkspaceDir) {
			return s.rollbackCheckpoint(&cp, true, true)
		}
	}
	return "", errors.New("no completed checkpoint owned by this session")
}

// RollbackCheckpoint restores only a completed checkpoint's run-owned
// manifest. Every current file must still match its recorded after hash; a
// mismatch is reported as a conflict and left untouched.
func (s *Session) RollbackCheckpoint(id string, approved bool) (string, error) {
	if !approved {
		return "", ErrRollbackApprovalRequired
	}
	checkpoints, err := s.ListCheckpoints()
	if err != nil {
		return "", err
	}
	for i := range checkpoints {
		if checkpoints[i].ID == id {
			if err := validateCheckpointIdentity(s, &checkpoints[i]); err != nil {
				return "", err
			}
			return s.rollbackCheckpoint(&checkpoints[i], true, false)
		}
	}
	return "", fmt.Errorf("checkpoint %q not found", id)
}

// PreviewRollbackCheckpoint performs the same ownership/hash checks as
// rollback, but never mutates files or history.
func (s *Session) PreviewRollbackCheckpoint(id string) (*RollbackPlan, error) {
	checkpoints, err := s.ListCheckpoints()
	if err != nil {
		return nil, err
	}
	for i := range checkpoints {
		if checkpoints[i].ID == id {
			if err := validateCheckpointIdentity(s, &checkpoints[i]); err != nil {
				return nil, err
			}
			return s.rollbackPlan(&checkpoints[i])
		}
	}
	return nil, fmt.Errorf("checkpoint %q not found", id)
}

// RollbackLastTurnWithApproval makes the approval boundary explicit for
// callers that prefer a named method over the compatibility variadic form.
func (s *Session) RollbackLastTurnWithApproval(approved bool) (string, error) {
	return s.RollbackLastTurn(approved)
}

// PreviewRollbackLastTurn returns the latest checkpoint plan without changing
// files or conversation history.
func (s *Session) PreviewRollbackLastTurn() (*RollbackPlan, error) {
	checkpoints, err := s.ListCheckpoints()
	if err != nil {
		return nil, err
	}
	for _, cp := range checkpoints {
		if cp.Completed && cp.SessionID == s.ID && sameWorkspace(cp.WorkspaceDir, s.WorkspaceDir) {
			return s.rollbackPlan(&cp)
		}
	}
	return nil, errors.New("no checkpoint owned by this session")
}

func (s *Session) rollbackCheckpoint(cp *Checkpoint, approved, removeHistory bool) (string, error) {
	plan, err := s.rollbackPlan(cp)
	if err != nil {
		return "", err
	}
	if len(plan.Conflicts) > 0 {
		return "", fmt.Errorf("rollback %s refused: user changes detected in %s", cp.ID, strings.Join(plan.Conflicts, ", "))
	}
	if !approved {
		return "", ErrRollbackApprovalRequired
	}
	workspace := cp.WorkspaceDir
	// Stage every baseline and re-check every after hash before changing any
	// path. A missing/corrupt blob therefore cannot leave a half-restored run.
	staged := make(map[string][]byte, len(plan.Restore))
	for _, entry := range cp.Manifest {
		if !contains(plan.Restore, entry.Path) && !contains(plan.Remove, entry.Path) {
			continue
		}
		current, exists, err := readWorkspaceFile(workspace, entry.Path)
		if err != nil {
			return "", fmt.Errorf("rollback %s inspect %s: %w", cp.ID, entry.Path, err)
		}
		if !snapshotMatches(current, exists, entry.AfterHash, entry.AfterExist) {
			return "", fmt.Errorf("rollback %s aborted: %s changed after confirmation", cp.ID, entry.Path)
		}
		if !entry.BeforeExist {
			continue
		}
		data, err := readBlob(filepath.Join(workspace, ".mncode", "checkpoints"), entry.BeforeBlob)
		if err != nil {
			return "", fmt.Errorf("rollback %s read baseline %s: %w", cp.ID, entry.Path, err)
		}
		if contentHash(data) != entry.BeforeHash {
			return "", fmt.Errorf("rollback %s baseline hash mismatch for %s", cp.ID, entry.Path)
		}
		staged[entry.Path] = data
	}
	for _, entry := range cp.Manifest {
		if data, ok := staged[entry.Path]; ok {
			if err := safeWriteWorkspaceFile(workspace, entry.Path, data, entry.BeforeMode); err != nil {
				return "", fmt.Errorf("rollback %s restore %s: %w", cp.ID, entry.Path, err)
			}
		} else if contains(plan.Remove, entry.Path) {
			if err := safeRemoveWorkspaceFile(workspace, entry.Path); err != nil {
				return "", fmt.Errorf("rollback %s remove %s: %w", cp.ID, entry.Path, err)
			}
		}
	}
	if removeHistory {
		popped := 0
		for len(s.History) > 0 {
			last := s.History[len(s.History)-1]
			s.History = s.History[:len(s.History)-1]
			popped++
			if last.Role == "user" {
				break
			}
		}
		if err := s.Save(); err != nil {
			return "", fmt.Errorf("rollback %s restored files but could not save history: %w", cp.ID, err)
		}
		return fmt.Sprintf("successfully rolled back %s (restored %d file(s), removed %d conversation message(s))", cp.ID, len(plan.Restore), popped), nil
	}
	return fmt.Sprintf("successfully rolled back %s (restored %d file(s))", cp.ID, len(plan.Restore)), nil
}

func (s *Session) rollbackPlan(cp *Checkpoint) (*RollbackPlan, error) {
	if !cp.Completed || len(cp.Manifest) == 0 {
		return nil, fmt.Errorf("checkpoint %s has no completed run-owned manifest", cp.ID)
	}
	plan := &RollbackPlan{CheckpointID: cp.ID}
	for _, entry := range cp.Manifest {
		if err := validateManifestPath(entry.Path); err != nil {
			return nil, fmt.Errorf("checkpoint %s invalid manifest: %w", cp.ID, err)
		}
		if !entry.Owned {
			plan.Skipped = append(plan.Skipped, entry.Path)
			continue
		}
		current, exists, err := readWorkspaceFile(cp.WorkspaceDir, entry.Path)
		if err != nil {
			return nil, fmt.Errorf("checkpoint %s inspect %s: %w", cp.ID, entry.Path, err)
		}
		if !snapshotMatches(current, exists, entry.AfterHash, entry.AfterExist) {
			// Any divergence from the recorded after-state is a user-change
			// conflict, including a file removed after the run.
			plan.Conflicts = append(plan.Conflicts, entry.Path)
			continue
		}
		if entry.BeforeExist {
			if !snapshotMatches(current, exists, entry.BeforeHash, true) {
				plan.Restore = append(plan.Restore, entry.Path)
			} else {
				plan.Skipped = append(plan.Skipped, entry.Path)
			}
		} else if exists {
			plan.Remove = append(plan.Remove, entry.Path)
		}
	}
	return plan, nil
}

// ListCheckpoints lists valid metadata records newest first. Malformed records
// are ignored so one interrupted write cannot hide the rest of the history.
func (s *Session) ListCheckpoints() ([]Checkpoint, error) {
	workspace, err := checkpointWorkspace(s.WorkspaceDir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(workspace, ".mncode", "checkpoints"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	list := make([]Checkpoint, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(workspace, ".mncode", "checkpoints", e.Name()))
		if err != nil {
			continue
		}
		var cp Checkpoint
		if json.Unmarshal(data, &cp) == nil {
			list = append(list, cp)
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Timestamp.After(list[j].Timestamp) })
	return list, nil
}

type fileSnapshot struct {
	Data []byte
	Hash string
	Mode uint32
}

func captureWorkspace(workspace, checkpointDir string) (map[string]fileSnapshot, error) {
	result := make(map[string]fileSnapshot)
	err := filepath.WalkDir(workspace, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(workspace, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) || rel == ".mncode" || strings.HasPrefix(rel, ".mncode"+string(filepath.Separator)) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		result[filepath.ToSlash(rel)] = fileSnapshot{Data: data, Hash: hex.EncodeToString(hash[:]), Mode: uint32(info.Mode().Perm())}
		return nil
	})
	return result, err
}

func saveBlob(checkpointDir string, snap fileSnapshot) (string, error) {
	name := filepath.Join("blobs", snap.Hash)
	path := filepath.Join(checkpointDir, name)
	if _, err := os.Stat(path); err == nil {
		return filepath.ToSlash(name), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := writePrivateAtomic(path, snap.Data, 0o600); err != nil {
		return "", err
	}
	return filepath.ToSlash(name), nil
}

func persistCheckpoint(cp *Checkpoint, checkpointDir string) error {
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("encode checkpoint: %w", err)
	}
	if err := writePrivateAtomic(filepath.Join(checkpointDir, cp.ID+".json"), data, 0o600); err != nil {
		return fmt.Errorf("write checkpoint metadata: %w", err)
	}
	patch, err := json.Marshal(struct {
		Version int              `json:"version"`
		ID      string           `json:"checkpoint_id"`
		Files   []CheckpointFile `json:"files"`
	}{1, cp.ID, cp.Manifest})
	if err != nil {
		return fmt.Errorf("encode checkpoint patch: %w", err)
	}
	cp.PatchFile = filepath.Join(checkpointDir, cp.ID+".patch")
	if err := writePrivateAtomic(cp.PatchFile, patch, 0o600); err != nil {
		return fmt.Errorf("write checkpoint patch: %w", err)
	}
	// Metadata needs the patch path, so update it once more atomically.
	data, err = json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("encode checkpoint metadata: %w", err)
	}
	return writePrivateAtomic(filepath.Join(checkpointDir, cp.ID+".json"), data, 0o600)
}

func validateCheckpointIdentity(s *Session, cp *Checkpoint) error {
	if cp == nil {
		return errors.New("nil checkpoint")
	}
	if cp.SessionID != s.ID || !sameWorkspace(cp.WorkspaceDir, s.WorkspaceDir) {
		return fmt.Errorf("checkpoint %s belongs to another session or workspace", cp.ID)
	}
	return nil
}

func checkpointWorkspace(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("checkpoint workspace is empty")
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", fmt.Errorf("resolve checkpoint workspace: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat checkpoint workspace: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("checkpoint workspace %q is not a directory", raw)
	}
	return filepath.EvalSymlinks(abs)
}

func sameWorkspace(a, b string) bool {
	aa, errA := checkpointWorkspace(a)
	bb, errB := checkpointWorkspace(b)
	return errA == nil && errB == nil && aa == bb
}

func cleanCheckpointPath(raw string) (string, error) {
	path := filepath.ToSlash(filepath.Clean(raw))
	if path == "." || filepath.IsAbs(raw) || path == ".." || strings.HasPrefix(path, "../") {
		return "", fmt.Errorf("checkpoint path %q escapes workspace", raw)
	}
	return path, nil
}

func validateManifestPath(path string) error {
	_, err := cleanCheckpointPath(path)
	return err
}

func readWorkspaceFile(workspace, rel string) ([]byte, bool, error) {
	if err := validateManifestPath(rel); err != nil {
		return nil, false, err
	}
	path, err := safeWorkspacePath(workspace, rel)
	if err != nil {
		return nil, false, err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("workspace path is not a regular file")
	}
	data, err := os.ReadFile(path)
	return data, err == nil, err
}

func safeWorkspacePath(workspace, rel string) (string, error) {
	root, err := checkpointWorkspace(workspace)
	if err != nil {
		return "", err
	}
	clean, err := cleanCheckpointPath(rel)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, filepath.FromSlash(clean))
	parent := filepath.Dir(path)
	for parent != root {
		info, err := os.Lstat(parent)
		if os.IsNotExist(err) {
			parent = filepath.Dir(parent)
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("path parent is a symlink")
		}
		parent = filepath.Dir(parent)
	}
	return path, nil
}

func safeWriteWorkspaceFile(workspace, rel string, data []byte, mode uint32) error {
	path, err := safeWorkspacePath(workspace, rel)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return errors.New("refusing to overwrite symlink")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writePrivateAtomic(path, data, os.FileMode(mode)&0o777)
}

func safeRemoveWorkspaceFile(workspace, rel string) error {
	path, err := safeWorkspacePath(workspace, rel)
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("refusing to remove non-regular workspace path")
	}
	return os.Remove(path)
}

func readBlob(checkpointDir, rel string) ([]byte, error) {
	if rel == "" || filepath.IsAbs(rel) {
		return nil, errors.New("invalid checkpoint blob path")
	}
	cleanRel := filepath.Clean(filepath.FromSlash(rel))
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return nil, errors.New("checkpoint blob escapes checkpoint directory")
	}
	path := filepath.Join(checkpointDir, cleanRel)
	clean, err := filepath.Abs(path)
	if err != nil || !strings.HasPrefix(clean, filepath.Clean(checkpointDir)+string(filepath.Separator)) {
		return nil, errors.New("checkpoint blob escapes checkpoint directory")
	}
	info, err := os.Lstat(clean)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("checkpoint blob is not a regular file")
	}
	return os.ReadFile(clean)
}

func writePrivateAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".checkpoint-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return replaceExistingFile(tmpName, path)
}

func contentHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func snapshotMatches(data []byte, exists bool, expected string, expectedExists bool) bool {
	if exists != expectedExists {
		return false
	}
	if !exists {
		return true
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]) == expected
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
