package replay

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mncode/pkg/commandutil"
)

var ErrTraceLimit = errors.New("replay trace limit exceeded")

// Recorder appends synchronized, redacted lifecycle events to one trace.
type Recorder struct {
	mu      sync.Mutex
	store   *Store
	trace   Trace
	file    *os.File
	nextSeq int64
	bytes   int64
	closed  bool
}

// ID returns the immutable trace identifier.
func (r *Recorder) ID() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.trace.ID
}

// Record appends one bounded event and never stores raw images or secret keys.
func (r *Recorder) Record(kind Kind, turn int, value any) error {
	if r == nil {
		return errors.New("replay recorder is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recordLocked(kind, turn, value)
}

func (r *Recorder) recordLocked(kind Kind, turn int, value any) error {
	if r.closed {
		return errors.New("replay recorder is closed")
	}
	if r.store != nil && r.store.MaxEvents > 0 && r.trace.Events >= r.store.MaxEvents {
		return ErrTraceLimit
	}
	data, err := marshalSafe(value)
	if err != nil {
		return err
	}
	if len(data) > 256*1024 {
		return fmt.Errorf("%w: event payload too large", ErrTraceLimit)
	}
	event := Event{Seq: r.nextSeq, Kind: kind, At: time.Now().UTC(), Turn: turn, Data: data}
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	if r.store != nil && r.store.MaxBytes > 0 && r.bytes+int64(len(line)) > r.store.MaxBytes {
		return ErrTraceLimit
	}
	if _, err := r.file.Write(line); err != nil {
		return err
	}
	r.bytes += int64(len(line))
	r.nextSeq++
	r.trace.Events++
	return r.writeManifestLocked()
}

// RecordAgentEvent adapts the engine-neutral recorder interface.
func (r *Recorder) RecordAgentEvent(kind string, turn int, value any) error {
	return r.Record(Kind(kind), turn, value)
}

// Close finalizes the manifest; incomplete traces remain explicitly marked.
func (r *Recorder) Close(complete bool) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.trace.EndedAt = time.Now().UTC()
	r.trace.Complete = false
	sessionEndErr := r.recordLocked(KindSessionEnd, 0, map[string]bool{"complete": complete})
	r.closed = true
	if err := r.file.Sync(); err != nil {
		_ = r.file.Close()
		return err
	}
	if err := r.file.Close(); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(r.traceDir(), "trace.jsonl"))
	if err != nil {
		return err
	}
	r.trace.Checksum = checksum(data)
	if sessionEndErr != nil && !errors.Is(sessionEndErr, ErrTraceLimit) {
		return sessionEndErr
	}
	r.trace.Complete = complete
	return r.writeManifestLocked()
}

func (r *Recorder) writeManifestLocked() error {
	return commandutil.WritePrivateJSON(filepath.Join(r.traceDir(), "manifest.json"), r.trace)
}

func (r *Recorder) traceDir() string {
	return filepath.Join(r.store.Workspace.Root, r.store.Dir, r.trace.ID)
}

func checksum(data []byte) string    { return fmt.Sprintf("%x", sha256Sum(data)) }
func sha256Sum(data []byte) [32]byte { return sha256.Sum256(data) }
