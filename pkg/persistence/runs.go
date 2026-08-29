package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrLeaseHeld = errors.New("lease is already held by another unexpired owner")

// RunRecord, JobRecord, EventRecord, and LeaseRecord define durable contracts
// consumed by Phase 3. This package intentionally provides storage primitives,
// not orchestration, retries, cancellation, or worker behavior.
type RunRecord struct {
	ID         string          `json:"id"`
	SessionID  string          `json:"sessionId,omitempty"`
	Profile    Profile         `json:"profile"`
	Status     string          `json:"status"`
	Model      string          `json:"model,omitempty"`
	Provider   string          `json:"provider,omitempty"`
	StartedAt  time.Time       `json:"startedAt"`
	FinishedAt *time.Time      `json:"finishedAt,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}
type JobRecord struct {
	ID        string          `json:"id"`
	RunID     string          `json:"runId"`
	Status    string          `json:"status"`
	Kind      string          `json:"kind,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}
type EventRecord struct {
	ID        int64           `json:"id,omitempty"`
	RunID     string          `json:"runId"`
	JobID     string          `json:"jobId,omitempty"`
	Sequence  int             `json:"sequence"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
}
type LeaseRecord struct {
	ID         string    `json:"id"`
	RunID      string    `json:"runId"`
	Holder     string    `json:"holder"`
	ExpiresAt  time.Time `json:"expiresAt"`
	AcquiredAt time.Time `json:"acquiredAt"`
}

type RunStore interface {
	SaveRun(context.Context, RunRecord) error
	GetRun(context.Context, string) (RunRecord, error)
	SaveJob(context.Context, JobRecord) error
	AppendEvent(context.Context, EventRecord) error
	ListEvents(context.Context, string, int) ([]EventRecord, error)
	AcquireLease(context.Context, LeaseRecord) error
	ReleaseLease(context.Context, string, string) error
}

func (s *Store) SaveRun(ctx context.Context, r RunRecord) error {
	if r.ID == "" || r.Status == "" {
		return fmt.Errorf("run id and status are required")
	}
	if r.StartedAt.IsZero() {
		r.StartedAt = time.Now().UTC()
	}
	var finished any
	if r.FinishedAt != nil {
		finished = r.FinishedAt.UnixNano()
	}
	return s.execRetry(ctx, `INSERT INTO runs(id,session_id,workspace_dir,profile_id,chat_id,status,model,provider,started_at,finished_at,metadata) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET session_id=excluded.session_id,workspace_dir=excluded.workspace_dir,profile_id=excluded.profile_id,chat_id=excluded.chat_id,status=excluded.status,model=excluded.model,provider=excluded.provider,started_at=excluded.started_at,finished_at=excluded.finished_at,metadata=excluded.metadata`, r.ID, r.SessionID, r.Profile.WorkspaceDir, r.Profile.ProfileID, r.Profile.ChatID, r.Status, r.Model, r.Provider, r.StartedAt.UnixNano(), finished, []byte(r.Metadata))
}
func (s *Store) GetRun(ctx context.Context, id string) (RunRecord, error) {
	var r RunRecord
	var started int64
	var finished sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT id,session_id,workspace_dir,profile_id,chat_id,status,model,provider,started_at,finished_at,metadata FROM runs WHERE id=?`, id).Scan(&r.ID, &r.SessionID, &r.Profile.WorkspaceDir, &r.Profile.ProfileID, &r.Profile.ChatID, &r.Status, &r.Model, &r.Provider, &started, &finished, &r.Metadata)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	r.StartedAt = time.Unix(0, started).UTC()
	if finished.Valid {
		t := time.Unix(0, finished.Int64).UTC()
		r.FinishedAt = &t
	}
	return r, nil
}
func (s *Store) SaveJob(ctx context.Context, j JobRecord) error {
	if j.ID == "" || j.RunID == "" || j.Status == "" {
		return fmt.Errorf("job id, run id, and status are required")
	}
	if j.CreatedAt.IsZero() {
		j.CreatedAt = time.Now().UTC()
	}
	if j.UpdatedAt.IsZero() {
		j.UpdatedAt = j.CreatedAt
	}
	return s.execRetry(ctx, `INSERT INTO jobs(id,run_id,status,kind,created_at,updated_at,metadata) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET run_id=excluded.run_id,status=excluded.status,kind=excluded.kind,updated_at=excluded.updated_at,metadata=excluded.metadata`, j.ID, j.RunID, j.Status, j.Kind, j.CreatedAt.UnixNano(), j.UpdatedAt.UnixNano(), []byte(j.Metadata))
}
func (s *Store) AppendEvent(ctx context.Context, e EventRecord) error {
	if e.RunID == "" || e.Type == "" {
		return fmt.Errorf("event run id and type are required")
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	return s.execRetry(ctx, `INSERT INTO events(run_id,job_id,sequence,type,payload,created_at) VALUES(?,?,?,?,?,?)`, e.RunID, e.JobID, e.Sequence, e.Type, []byte(e.Payload), e.CreatedAt.UnixNano())
}
func (s *Store) ListEvents(ctx context.Context, runID string, limit int) ([]EventRecord, error) {
	q := `SELECT id,run_id,job_id,sequence,type,payload,created_at FROM events WHERE run_id=? ORDER BY sequence`
	a := []any{runID}
	if limit > 0 {
		q += ` LIMIT ?`
		a = append(a, limit)
	}
	rows, err := s.db.QueryContext(ctx, q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventRecord
	for rows.Next() {
		var e EventRecord
		var job sql.NullString
		var created int64
		if err := rows.Scan(&e.ID, &e.RunID, &job, &e.Sequence, &e.Type, &e.Payload, &created); err != nil {
			return nil, err
		}
		e.JobID = job.String
		e.CreatedAt = time.Unix(0, created).UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}
func (s *Store) AcquireLease(ctx context.Context, l LeaseRecord) error {
	if l.ID == "" || l.RunID == "" || l.Holder == "" {
		return fmt.Errorf("lease id, run id, and holder are required")
	}
	if l.AcquiredAt.IsZero() {
		l.AcquiredAt = time.Now().UTC()
	}
	if l.ExpiresAt.IsZero() {
		return fmt.Errorf("lease expiry is required")
	}
	tx, err := s.beginRetry(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var existingHolder string
	var expiresAt int64
	err = tx.QueryRowContext(ctx, `SELECT holder, expires_at FROM leases WHERE id = ?`, l.ID).Scan(&existingHolder, &expiresAt)
	if err == nil {
		nowNano := l.AcquiredAt.UnixNano()
		if expiresAt > nowNano && existingHolder != l.Holder {
			return ErrLeaseHeld
		}
		_, err = tx.ExecContext(ctx, `UPDATE leases SET run_id = ?, holder = ?, expires_at = ?, acquired_at = ? WHERE id = ?`,
			l.RunID, l.Holder, l.ExpiresAt.UnixNano(), l.AcquiredAt.UnixNano(), l.ID)
		if err != nil {
			return err
		}
		return tx.Commit()
	} else if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, `INSERT INTO leases(id, run_id, holder, expires_at, acquired_at) VALUES(?, ?, ?, ?, ?)`,
			l.ID, l.RunID, l.Holder, l.ExpiresAt.UnixNano(), l.AcquiredAt.UnixNano())
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	return err
}
func (s *Store) ReleaseLease(ctx context.Context, id, holder string) error {
	if id == "" || holder == "" {
		return fmt.Errorf("lease id and holder are required")
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM leases WHERE id=? AND holder=?`, id, holder)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
