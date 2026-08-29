package persistence

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), StoreConfig{Path: filepath.Join(t.TempDir(), "profile.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSchemaMigrationAndIdempotence(t *testing.T) {
	s := testStore(t)
	version, err := s.SchemaVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if version != CurrentSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, CurrentSchemaVersion)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.DB().QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('sessions','messages','runs','jobs','events','leases','migration_markers')`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 7 {
		t.Fatalf("canonical table count = %d, want 7", count)
	}
	var mode string
	if err := s.DB().QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Fatalf("journal mode = %q, want wal", mode)
	}
}

func TestSessionJSONRoundTripAndFTS(t *testing.T) {
	s := testStore(t)
	want := SessionRecord{ID: "chat-1", Title: "SQLite notes", WorkspaceDir: "/workspace", ProfileID: "profile-a", ChatID: "chat-1", Model: "model-a", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), Messages: []MessageRecord{{Role: "user", Content: "find this durable phrase", Payload: json.RawMessage(`{"role":"user","content":"find this durable phrase"}`)}}}
	if err := s.SaveSession(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSession(context.Background(), want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.Messages[0].Content != want.Messages[0].Content {
		t.Fatalf("round trip mismatch: %#v", got)
	}
	matches, err := s.SearchSessions(context.Background(), SearchFilter{Content: "durable phrase", ProfileID: "profile-a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].ID != want.ID {
		t.Fatalf("FTS matches = %#v", matches)
	}
	data, err := ExportSessionJSON(context.Background(), s, want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportSessionJSON(context.Background(), s, data); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyImportIsCopyOnWriteAndIdempotent(t *testing.T) {
	s := testStore(t)
	data := []byte(`{"id":"legacy-1","title":"Legacy","workspaceDir":"/tmp/work","messages":[{"role":"user","content":"hello"}]}`)
	r, first, err := ImportLegacySessionJSON(context.Background(), s, data, "/tmp/legacy.backup")
	if err != nil {
		t.Fatal(err)
	}
	if first.AlreadyImported || first.Marker.Status != "complete" {
		t.Fatalf("first import report = %#v", first)
	}
	if string(data) != `{"id":"legacy-1","title":"Legacy","workspaceDir":"/tmp/work","messages":[{"role":"user","content":"hello"}]}` {
		t.Fatal("legacy source was modified")
	}
	_, second, err := ImportLegacySessionJSON(context.Background(), s, data, "/tmp/legacy.backup")
	if err != nil {
		t.Fatal(err)
	}
	if !second.AlreadyImported {
		t.Fatal("second import was not idempotent")
	}
	if second.Marker.SourceFingerprint != first.Marker.SourceFingerprint {
		t.Fatal("source fingerprint changed")
	}
	if r.Messages[0].Payload == nil {
		t.Fatal("legacy message payload was not retained")
	}
	var count int
	if err := s.DB().QueryRow(`SELECT count(*) FROM sessions WHERE id='legacy-1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("imported session rows = %d", count)
	}
}

func TestPrivateDatabasePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "state.db")
	s, err := Open(context.Background(), StoreConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	for _, item := range []string{filepath.Dir(path), path} {
		st, err := os.Stat(item)
		if err != nil {
			t.Fatal(err)
		}
		want := os.FileMode(0o700)
		if item == path {
			want = 0o600
		}
		if got := st.Mode().Perm(); got != want {
			t.Fatalf("%s mode = %04o, want %04o", item, got, want)
		}
	}
}
