package memory

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Entry struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Source    string `json:"source,omitempty"`
	CreatedAt string `json:"createdAt"`
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mncode", "memories.json"), nil
}

func Load() ([]Entry, error) {
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
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func Add(text, source string) (Entry, error) {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return Entry{}, errors.New("memory text cannot be empty")
	}
	entries, err := Load()
	if err != nil {
		return Entry{}, err
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Text, clean) {
			return entry, nil
		}
	}
	entry := Entry{
		ID:        time.Now().UTC().Format("20060102T150405.000000000Z"),
		Text:      clean,
		Source:    strings.TrimSpace(source),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	entries = append(entries, entry)
	if len(entries) > 100 {
		entries = entries[len(entries)-100:]
	}
	return entry, save(entries)
}

func Clear() (int, error) {
	path, err := Path()
	if err != nil {
		return 0, err
	}
	entries, err := Load()
	if err != nil {
		return 0, err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	return len(entries), nil
}

func save(entries []Entry) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
