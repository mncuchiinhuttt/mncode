// Package persistence owns the canonical local state database used by CLI and Desktop.
// It deliberately keeps import/export adapters separate from the legacy JSON files so
// a failed migration can never destroy the source of truth.
package persistence

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	CurrentSchemaVersion = 1
	defaultBusyRetries   = 5
)

var ErrSQLiteUnavailable = errors.New("canonical persistence SQLite is unavailable")
var ErrNotFound = errors.New("persistence record not found")

// sqliteUnavailable marks failures that prevent the canonical database from
// being opened. Callers that retain a legacy JSON path can safely fall back
// when this sentinel is present.
func sqliteUnavailable(err error) error {
	if err == nil || errors.Is(err, ErrSQLiteUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrSQLiteUnavailable, err)
}

// Profile identifies the isolation boundary shared by CLI and Desktop.
type Profile struct {
	WorkspaceDir string
	ProfileID    string
	ChatID       string
	RunID        string
}

// SessionRecord is the driver-neutral canonical representation of a chat.
// Payload on each message contains the source provider's complete JSON message,
// allowing adapters to retain fields added by newer providers.
type SessionRecord struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	WorkspaceDir string          `json:"workspaceDir"`
	ProfileID    string          `json:"profileId"`
	ChatID       string          `json:"chatId"`
	RunID        string          `json:"runId,omitempty"`
	Model        string          `json:"model"`
	Provider     string          `json:"provider,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	Turns        int             `json:"turns"`
	Messages     []MessageRecord `json:"messages"`
}

type MessageRecord struct {
	ID        string          `json:"id,omitempty"`
	Sequence  int             `json:"sequence"`
	Role      string          `json:"role"`
	Content   string          `json:"content,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	CreatedAt time.Time       `json:"createdAt,omitempty"`
}

type SearchFilter struct {
	ProfileID string
	Workspace string
	ChatID    string
	Model     string
	Provider  string
	Title     string
	Content   string
	Since     time.Time
	Until     time.Time
	Limit     int
}

type StoreConfig struct {
	Path        string
	Profile     string
	BusyRetries int
	BusyDelay   time.Duration
}

type MigrationMarker struct {
	ID                string    `json:"id"`
	SourceFingerprint string    `json:"sourceFingerprint"`
	Status            string    `json:"status"`
	BackupPath        string    `json:"backupPath,omitempty"`
	SourceCount       int       `json:"sourceCount"`
	ImportedCount     int       `json:"importedCount"`
	SourceHash        string    `json:"sourceHash"`
	ImportedHash      string    `json:"importedHash"`
	StartedAt         time.Time `json:"startedAt"`
	CompletedAt       time.Time `json:"completedAt,omitempty"`
}

type Store struct {
	db          *sql.DB
	busyRetries int
	busyDelay   time.Duration
	mu          sync.RWMutex
}
type SessionStore interface {
	SaveSession(context.Context, SessionRecord) error
	GetSession(context.Context, string) (SessionRecord, error)
	ListSessions(context.Context, SearchFilter) ([]SessionRecord, error)
	SearchSessions(context.Context, SearchFilter) ([]SessionRecord, error)
}

// DefaultPath returns the profile database under ~/.mncode with restrictive
// directory permissions. Profile IDs are sanitized to prevent path traversal.
func DefaultPath(profile string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	profile = strings.TrimSpace(profile)
	if profile == "" {
		profile = "default"
	}

	if filepath.Base(profile) != profile || strings.Contains(profile, "..") || strings.ContainsAny(profile, `/\\`) {
		return "", fmt.Errorf("invalid profile id")
	}
	return filepath.Join(home, ".mncode", "state", profile+".db"), nil
}

func NewStore(ctx context.Context, cfg StoreConfig) (*Store, error) { return Open(ctx, cfg) }

// Open opens (and migrates) the canonical database. A database opened by this
// package always uses WAL and a bounded busy timeout/retry policy.
func Open(ctx context.Context, cfg StoreConfig) (*Store, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.Path == "" {
		path, err := DefaultPath(cfg.Profile)
		if err != nil {
			return nil, sqliteUnavailable(err)
		}
		cfg.Path = path
	}
	if cfg.BusyRetries <= 0 {
		cfg.BusyRetries = defaultBusyRetries
	}
	if cfg.BusyDelay <= 0 {
		cfg.BusyDelay = 10 * time.Millisecond
	}
	if err := ensurePrivatePath(cfg.Path); err != nil {
		return nil, sqliteUnavailable(err)
	}
	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, sqliteUnavailable(err)
	}
	s := &Store{db: db, busyRetries: cfg.BusyRetries, busyDelay: cfg.BusyDelay}
	if err = s.configure(ctx); err != nil {
		_ = db.Close()
		return nil, sqliteUnavailable(err)
	}
	if err = s.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, sqliteUnavailable(err)
	}
	if err = os.Chmod(cfg.Path, 0o600); err != nil {
		_ = db.Close()
		return nil, sqliteUnavailable(err)
	}
	return s, nil
}

func ensurePrivatePath(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	if st, err := os.Stat(path); err == nil {
		if st.IsDir() {
			return fmt.Errorf("database path is a directory")
		}
		if err := os.Chmod(path, 0o600); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Store) configure(ctx context.Context) error {
	for _, q := range []string{"PRAGMA journal_mode=WAL", "PRAGMA synchronous=NORMAL", "PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=250"} {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("configure sqlite: %w", err)
		}
	}
	return nil
}

func (s *Store) DB() *sql.DB { return s.db }
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS sessions (id TEXT PRIMARY KEY, workspace_dir TEXT NOT NULL DEFAULT '', profile_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', run_id TEXT NOT NULL DEFAULT '', title TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT '', provider TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, turns INTEGER NOT NULL DEFAULT 0)`,
		`CREATE INDEX IF NOT EXISTS sessions_identity_idx ON sessions(profile_id, workspace_dir, chat_id, updated_at DESC)`,
		`CREATE TABLE IF NOT EXISTS messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE, sequence INTEGER NOT NULL, role TEXT NOT NULL, content TEXT NOT NULL DEFAULT '', thinking TEXT NOT NULL DEFAULT '', payload BLOB, created_at INTEGER NOT NULL, UNIQUE(session_id, sequence))`,
		`CREATE INDEX IF NOT EXISTS messages_session_idx ON messages(session_id, sequence)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS message_fts USING fts5(session_id UNINDEXED, message_id UNINDEXED, content, role)`,
		`CREATE TABLE IF NOT EXISTS tool_calls (id TEXT PRIMARY KEY, session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE, message_id TEXT, name TEXT NOT NULL, arguments BLOB, result BLOB, is_error INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS usage (id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE, input_tokens INTEGER NOT NULL DEFAULT 0, output_tokens INTEGER NOT NULL DEFAULT 0, thinking_tokens INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS runs (id TEXT PRIMARY KEY, session_id TEXT, workspace_dir TEXT NOT NULL DEFAULT '', profile_id TEXT NOT NULL DEFAULT '', chat_id TEXT NOT NULL DEFAULT '', status TEXT NOT NULL, model TEXT NOT NULL DEFAULT '', provider TEXT NOT NULL DEFAULT '', started_at INTEGER NOT NULL, finished_at INTEGER, metadata BLOB)`,
		`CREATE TABLE IF NOT EXISTS jobs (id TEXT PRIMARY KEY, run_id TEXT NOT NULL, status TEXT NOT NULL, kind TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, metadata BLOB)`,
		`CREATE TABLE IF NOT EXISTS events (id INTEGER PRIMARY KEY AUTOINCREMENT, run_id TEXT NOT NULL, job_id TEXT, sequence INTEGER NOT NULL, type TEXT NOT NULL, payload BLOB, created_at INTEGER NOT NULL, UNIQUE(run_id, sequence))`,
		`CREATE TABLE IF NOT EXISTS leases (id TEXT PRIMARY KEY, run_id TEXT NOT NULL, holder TEXT NOT NULL, expires_at INTEGER NOT NULL, acquired_at INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS migration_markers (id TEXT PRIMARY KEY, source_fingerprint TEXT NOT NULL, status TEXT NOT NULL, backup_path TEXT NOT NULL DEFAULT '', source_count INTEGER NOT NULL DEFAULT 0, imported_count INTEGER NOT NULL DEFAULT 0, source_hash TEXT NOT NULL DEFAULT '', imported_hash TEXT NOT NULL DEFAULT '', started_at INTEGER NOT NULL, completed_at INTEGER)`,
	}
	for _, stmt := range stmts {
		if err := s.execRetry(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	if err := s.execRetry(ctx, `INSERT INTO schema_meta(key,value) VALUES('schema_version',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, CurrentSchemaVersion); err != nil {
		return err
	}
	return nil
}

func (s *Store) SchemaVersion(ctx context.Context) (int, error) {
	var v int
	err := s.db.QueryRowContext(ctx, `SELECT value FROM schema_meta WHERE key='schema_version'`).Scan(&v)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (s *Store) SaveSession(ctx context.Context, rec SessionRecord) error {
	if rec.ID == "" {
		return fmt.Errorf("session id is required")
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	if rec.UpdatedAt.IsZero() {
		rec.UpdatedAt = rec.CreatedAt
	}
	if rec.Turns == 0 {
		for _, m := range rec.Messages {
			if m.Role == "user" {
				rec.Turns++
			}
		}
	}
	tx, err := s.beginRetry(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO sessions(id,workspace_dir,profile_id,chat_id,run_id,title,model,provider,created_at,updated_at,turns) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET workspace_dir=excluded.workspace_dir,profile_id=excluded.profile_id,chat_id=excluded.chat_id,run_id=excluded.run_id,title=excluded.title,model=excluded.model,provider=excluded.provider,updated_at=excluded.updated_at,turns=excluded.turns`, rec.ID, rec.WorkspaceDir, rec.ProfileID, rec.ChatID, rec.RunID, rec.Title, rec.Model, rec.Provider, rec.CreatedAt.UnixNano(), rec.UpdatedAt.UnixNano(), rec.Turns)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM messages WHERE session_id=?`, rec.ID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM message_fts WHERE session_id=?`, rec.ID); err != nil {
		return err
	}
	for i, m := range rec.Messages {
		if m.ID == "" {
			m.ID = fmt.Sprintf("%s:%d", rec.ID, i)
		}
		if m.CreatedAt.IsZero() {
			m.CreatedAt = rec.UpdatedAt
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO messages(id,session_id,sequence,role,content,thinking,payload,created_at) VALUES(?,?,?,?,?,?,?,?)`, m.ID, rec.ID, i, m.Role, m.Content, m.Thinking, []byte(m.Payload), m.CreatedAt.UnixNano()); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO message_fts(session_id,message_id,content,role) VALUES(?,?,?,?)`, rec.ID, m.ID, m.Content, m.Role); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetSession(ctx context.Context, id string) (SessionRecord, error) {
	var r SessionRecord
	var created, updated int64
	err := s.db.QueryRowContext(ctx, `SELECT id,title,workspace_dir,profile_id,chat_id,run_id,model,provider,created_at,updated_at,turns FROM sessions WHERE id=?`, id).Scan(&r.ID, &r.Title, &r.WorkspaceDir, &r.ProfileID, &r.ChatID, &r.RunID, &r.Model, &r.Provider, &created, &updated, &r.Turns)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	r.CreatedAt, r.UpdatedAt = time.Unix(0, created).UTC(), time.Unix(0, updated).UTC()
	rows, err := s.db.QueryContext(ctx, `SELECT id,sequence,role,content,thinking,payload,created_at FROM messages WHERE session_id=? ORDER BY sequence`, id)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var m MessageRecord
		var payload []byte
		var created int64
		if err = rows.Scan(&m.ID, &m.Sequence, &m.Role, &m.Content, &m.Thinking, &payload, &created); err != nil {
			return r, err
		}
		m.Payload = append(json.RawMessage(nil), payload...)
		m.CreatedAt = time.Unix(0, created).UTC()
		r.Messages = append(r.Messages, m)
	}
	if err = rows.Err(); err != nil {
		return r, err
	}
	return r, nil
}

func (s *Store) GetMigration(ctx context.Context, id string) (MigrationMarker, error) {
	var m MigrationMarker
	var started int64
	var completed sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT id,source_fingerprint,status,backup_path,source_count,imported_count,source_hash,imported_hash,started_at,completed_at FROM migration_markers WHERE id=?`, id).Scan(&m.ID, &m.SourceFingerprint, &m.Status, &m.BackupPath, &m.SourceCount, &m.ImportedCount, &m.SourceHash, &m.ImportedHash, &started, &completed)
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, err
	}
	m.StartedAt = time.Unix(0, started).UTC()
	if completed.Valid {
		m.CompletedAt = time.Unix(0, completed.Int64).UTC()
	}
	return m, nil
}

func (s *Store) ListSessions(ctx context.Context, filter SearchFilter) ([]SessionRecord, error) {
	q := `SELECT id FROM sessions WHERE 1=1`
	args := []any{}
	q, args = addFilter(q, args, filter, false)
	q += ` ORDER BY updated_at DESC`
	if filter.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionRecord
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		r, err := s.GetSession(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func addFilter(q string, args []any, f SearchFilter, fts bool) (string, []any) {
	if f.ProfileID != "" {
		q += " AND profile_id=?"
		args = append(args, f.ProfileID)
	}
	if f.Workspace != "" {
		q += " AND workspace_dir=?"
		args = append(args, f.Workspace)
	}
	if f.ChatID != "" {
		q += " AND chat_id=?"
		args = append(args, f.ChatID)
	}
	if f.Model != "" {
		q += " AND model=?"
		args = append(args, f.Model)
	}
	if f.Provider != "" {
		q += " AND provider=?"
		args = append(args, f.Provider)
	}
	if f.Title != "" {
		q += " AND title LIKE ?"
		args = append(args, "%"+f.Title+"%")
	}
	if !f.Since.IsZero() {
		q += " AND updated_at>=?"
		args = append(args, f.Since.UnixNano())
	}
	if !f.Until.IsZero() {
		q += " AND updated_at<=?"
		args = append(args, f.Until.UnixNano())
	}
	_ = fts
	return q, args
}

func (s *Store) SearchSessions(ctx context.Context, filter SearchFilter) ([]SessionRecord, error) {
	if strings.TrimSpace(filter.Content) == "" {
		return s.ListSessions(ctx, filter)
	}
	q := `SELECT DISTINCT s.id FROM sessions s JOIN message_fts f ON f.session_id=s.id WHERE f.message_fts MATCH ?`
	args := []any{filter.Content}
	q, args = addFilter(q, args, filter, true)
	q += ` ORDER BY s.updated_at DESC`
	if filter.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return s.searchContentFallback(ctx, filter)
	}
	defer rows.Close()
	var out []SessionRecord
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		r, err := s.GetSession(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return s.searchContentFallback(ctx, filter)
	}
	return out, nil
}

func (s *Store) searchContentFallback(ctx context.Context, filter SearchFilter) ([]SessionRecord, error) {
	q := `SELECT DISTINCT s.id FROM sessions s JOIN messages m ON m.session_id=s.id WHERE m.content LIKE ?`
	args := []any{"%" + filter.Content + "%"}
	q, args = addFilter(q, args, filter, false)
	q += ` ORDER BY s.updated_at DESC`
	if filter.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionRecord
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		r, err := s.GetSession(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) beginRetry(ctx context.Context) (*sql.Tx, error) {
	var tx *sql.Tx
	var err error
	for i := 0; i <= s.busyRetries; i++ {
		tx, err = s.db.BeginTx(ctx, nil)
		if err == nil {
			return tx, nil
		}
		if !isBusy(err) || i == s.busyRetries {
			break
		}
		if err = waitContext(ctx, s.busyDelay*time.Duration(1<<i)); err != nil {
			break
		}
	}
	return nil, err
}
func (s *Store) execRetry(ctx context.Context, q string, args ...any) error {
	var err error
	for i := 0; i <= s.busyRetries; i++ {
		_, err = s.db.ExecContext(ctx, q, args...)
		if err == nil {
			return nil
		}
		if !isBusy(err) || i == s.busyRetries {
			break
		}
		if err = waitContext(ctx, s.busyDelay*time.Duration(1<<i)); err != nil {
			break
		}
	}
	return err
}
func isBusy(err error) bool {
	v := strings.ToLower(err.Error())
	return strings.Contains(v, "busy") || strings.Contains(v, "locked")
}
func waitContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func Fingerprint(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func FingerprintReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *Store) PutMigration(ctx context.Context, m MigrationMarker) error {
	if m.StartedAt.IsZero() {
		m.StartedAt = time.Now().UTC()
	}
	var completed any
	if !m.CompletedAt.IsZero() {
		completed = m.CompletedAt.UnixNano()
	}
	return s.execRetry(ctx, `INSERT INTO migration_markers(id,source_fingerprint,status,backup_path,source_count,imported_count,source_hash,imported_hash,started_at,completed_at) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET source_fingerprint=excluded.source_fingerprint,status=excluded.status,backup_path=excluded.backup_path,source_count=excluded.source_count,imported_count=excluded.imported_count,source_hash=excluded.source_hash,imported_hash=excluded.imported_hash,started_at=excluded.started_at,completed_at=excluded.completed_at`, m.ID, m.SourceFingerprint, m.Status, m.BackupPath, m.SourceCount, m.ImportedCount, m.SourceHash, m.ImportedHash, m.StartedAt.UnixNano(), completed)
}

// Backup copies a source file without changing it and returns the backup path.
func Backup(source, destination string) (string, error) {
	if source == "" {
		return "", fmt.Errorf("source is required")
	}
	in, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer in.Close()
	if destination == "" {
		destination = source + ".backup"
	}
	if err := ensurePrivatePath(destination); err != nil {
		return "", err
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(out, in); err == nil {
		err = out.Sync()
	}
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	return destination, nil
}

// StableSessionHash is used by migration verification; ordering is explicit.
func StableSessionHash(sessions []SessionRecord) string {
	cp := append([]SessionRecord(nil), sessions...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].ID < cp[j].ID })
	b, _ := json.Marshal(cp)
	return Fingerprint(b)
}

func ExportSessionJSON(ctx context.Context, s *Store, id string) ([]byte, error) {
	r, err := s.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(r, "", "  ")
}
func ImportSessionJSON(ctx context.Context, s *Store, data []byte) (SessionRecord, error) {
	var r SessionRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return r, err
	}
	if r.ID == "" {
		return r, fmt.Errorf("session id is required")
	}
	if err := s.SaveSession(ctx, r); err != nil {
		return r, err
	}
	return r, nil
}
