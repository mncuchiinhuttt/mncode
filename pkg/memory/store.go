package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	maxEntries  = 100
	maxTextSize = 16 * 1024
	maxFileSize = 1 * 1024 * 1024
)

var (
	storeMu           sync.Mutex
	ErrUnsafeMemory   = errors.New("memory contains unsafe prompt instructions")
	ErrMemoryTooLarge = errors.New("memory exceeds the size limit")
)

type Entry struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Source    string `json:"source,omitempty"`
	CreatedAt string `json:"createdAt"`
}

// MemorySnapshot is a frozen, per-turn view of approved memories. Callers
// receive copies so mutations cannot alter the snapshot or the store.
type MemorySnapshot struct {
	Entries  []Entry   `json:"entries"`
	Version  string    `json:"version"`
	LoadedAt time.Time `json:"loadedAt"`
}

func (s MemorySnapshot) EntriesCopy() []Entry {
	return append([]Entry(nil), s.Entries...)
}

// LoadSnapshot reads memories once and returns a content-versioned snapshot.
func LoadSnapshot() (MemorySnapshot, error) {
	path, err := Path()
	if err != nil {
		return MemorySnapshot{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return MemorySnapshot{Entries: []Entry{}, Version: hashBytes(nil), LoadedAt: time.Now().UTC()}, nil
	}
	if err != nil {
		return MemorySnapshot{}, err
	}
	if len(data) > maxFileSize {
		return MemorySnapshot{}, ErrMemoryTooLarge
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return MemorySnapshot{}, err
	}
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
	return MemorySnapshot{
		Entries:  append([]Entry(nil), entries...),
		Version:  hashBytes(data),
		LoadedAt: time.Now().UTC(),
	}, nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mncode", "memories.json"), nil
}
func Load() ([]Entry, error) {
	snapshot, err := LoadSnapshot()
	if err != nil {
		return nil, err
	}
	return snapshot.EntriesCopy(), nil
}

func Add(text, source string) (Entry, error) {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return Entry{}, errors.New("memory text cannot be empty")
	}
	if len([]byte(clean)) > maxTextSize {
		return Entry{}, ErrMemoryTooLarge
	}
	if containsUnsafePrompt(clean) {
		return Entry{}, ErrUnsafeMemory
	}
	storeMu.Lock()
	defer storeMu.Unlock()
	entries, err := loadUnlocked()
	if err != nil {
		return Entry{}, err
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Text, clean) {
			return entry, nil
		}
	}
	now := time.Now().UTC()
	entry := Entry{
		ID:        now.Format("20060102T150405.000000000Z"),
		Text:      clean,
		Source:    strings.TrimSpace(source),
		CreatedAt: now.Format(time.RFC3339),
	}
	entries = append(entries, entry)
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
	return entry, saveUnlocked(entries)
}

func containsUnsafePrompt(text string) bool {
	lower := strings.ToLower(text)
	for _, marker := range []string{
		"ignore previous instructions", "ignore all prior instructions",
		"disregard the system prompt", "<system>", "<assistant>",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func loadUnlocked() ([]Entry, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > maxFileSize {
		return nil, ErrMemoryTooLarge
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func Clear() (int, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	entries, err := loadUnlocked()
	if err != nil {
		return 0, err
	}
	path, err := Path()
	if err != nil {
		return 0, err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	return len(entries), nil
}

func save(entries []Entry) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	return saveUnlocked(entries)
}

func saveUnlocked(entries []Entry) error {
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxFileSize {
		return ErrMemoryTooLarge
	}
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".memories-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
