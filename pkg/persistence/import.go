package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ImportReport is deliberately explicit so callers can keep source files until
// an operator has verified the migration.
type ImportReport struct {
	Marker          MigrationMarker `json:"marker"`
	SourcePath      string          `json:"sourcePath,omitempty"`
	AlreadyImported bool            `json:"alreadyImported"`
}

// ImportLegacySessionJSON imports one SavedSession-shaped JSON document. It is
// copy-on-write and idempotent by source fingerprint; the input bytes are never
// modified or removed.
func ImportLegacySessionJSON(ctx context.Context, s *Store, data []byte, backupPath string) (SessionRecord, ImportReport, error) {
	var raw struct {
		ID           string            `json:"id"`
		Title        string            `json:"title"`
		WorkspaceDir string            `json:"workspaceDir"`
		ProfileID    string            `json:"profileId"`
		ChatID       string            `json:"chatId"`
		RunID        string            `json:"runId"`
		Model        string            `json:"model"`
		Provider     string            `json:"provider"`
		CreatedAt    time.Time         `json:"createdAt"`
		UpdatedAt    time.Time         `json:"updatedAt"`
		Turns        int               `json:"turns"`
		Messages     []json.RawMessage `json:"messages"`
	}
	var rec SessionRecord
	if err := json.Unmarshal(data, &raw); err != nil {
		return rec, ImportReport{}, err
	}
	if raw.ID == "" {
		return rec, ImportReport{}, fmt.Errorf("session id is required")
	}
	rec = SessionRecord{ID: raw.ID, Title: raw.Title, WorkspaceDir: raw.WorkspaceDir, ProfileID: raw.ProfileID, ChatID: raw.ChatID, RunID: raw.RunID, Model: raw.Model, Provider: raw.Provider, CreatedAt: raw.CreatedAt, UpdatedAt: raw.UpdatedAt, Turns: raw.Turns}
	for i, b := range raw.Messages {
		var m struct {
			ID       string `json:"id"`
			Role     string `json:"role"`
			Content  string `json:"content"`
			Thinking string `json:"thinking"`
		}
		if err := json.Unmarshal(b, &m); err != nil {
			return rec, ImportReport{}, err
		}
		msgID := m.ID
		if msgID == "" {
			msgID = fmt.Sprintf("%s:msg:%d", raw.ID, i)
		}
		rec.Messages = append(rec.Messages, MessageRecord{ID: msgID, Sequence: i, Role: m.Role, Content: m.Content, Thinking: m.Thinking, Payload: append(json.RawMessage(nil), b...)})
	}
	finger := Fingerprint(data)
	markerID := "legacy-session:" + raw.ID + ":" + finger
	existing, err := s.GetMigration(ctx, markerID)
	if err == nil && existing.Status == "complete" {
		return rec, ImportReport{Marker: existing, AlreadyImported: true}, nil
	}
	if err != nil && err != ErrNotFound {
		return rec, ImportReport{}, err
	}
	m := MigrationMarker{ID: markerID, SourceFingerprint: finger, Status: "started", BackupPath: backupPath, SourceCount: 1, SourceHash: finger}
	if err := s.PutMigration(ctx, m); err != nil {
		return rec, ImportReport{}, err
	}
	if err := s.SaveSession(ctx, rec); err != nil {
		m.Status = "failed"
		_ = s.PutMigration(ctx, m)
		return rec, ImportReport{Marker: m}, err
	}
	m.Status = "complete"
	m.ImportedCount = 1
	m.ImportedHash = StableSessionHash([]SessionRecord{rec})
	m.CompletedAt = time.Now().UTC()
	if err := s.PutMigration(ctx, m); err != nil {
		return rec, ImportReport{Marker: m}, err
	}
	return rec, ImportReport{Marker: m}, nil
}

func ImportLegacySessionFile(ctx context.Context, s *Store, path string) (SessionRecord, ImportReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SessionRecord{}, ImportReport{SourcePath: path}, err
	}
	backup := path + ".backup"
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		if _, err = Backup(path, backup); err != nil {
			return SessionRecord{}, ImportReport{SourcePath: path}, err
		}
	}
	r, report, err := ImportLegacySessionJSON(ctx, s, data, backup)
	report.SourcePath = path
	return r, report, err
}

// MigrationManifestPath records the canonical location/version separately from
// database rows, making recovery tooling able to find the store before opening it.
func MigrationManifestPath(profile string) (string, error) {
	p, err := DefaultPath(profile)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p), "migration-manifest.json"), nil
}
func WriteMigrationManifest(profile string, version int) (string, error) {
	path, err := MigrationManifestPath(profile)
	if err != nil {
		return "", err
	}
	b, _ := json.MarshalIndent(map[string]any{"profile": profile, "schemaVersion": version, "databasePath": mustDefaultPath(profile), "updatedAt": time.Now().UTC()}, "", "  ")
	if err := writeAtomic(path, b); err != nil {
		return "", err
	}
	return path, nil
}
func mustDefaultPath(profile string) string { p, _ := DefaultPath(profile); return p }
func writeAtomic(path string, b []byte) error {
	if err := ensurePrivatePath(path); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".manifest-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err = tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(b)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
