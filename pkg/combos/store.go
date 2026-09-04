package combos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Store manages loading, saving, and querying agent combos.
type Store struct {
	mu           sync.RWMutex
	combos       map[string]*Combo
	workspaceDir string
	globalDir    string
}

// NewStore initializes a combo store with global and workspace-scoped paths.
func NewStore(workspaceDir string) (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	globalDir := filepath.Join(home, ".mncode", "combos")
	if err := os.MkdirAll(globalDir, 0700); err != nil {
		return nil, fmt.Errorf("create global combos dir: %w", err)
	}

	s := &Store{
		combos:       make(map[string]*Combo),
		workspaceDir: workspaceDir,
		globalDir:    globalDir,
	}
	_ = s.Load()
	return s, nil
}

// Load reads built-in presets, global combos, and workspace combos.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.combos = make(map[string]*Combo)

	// 1. Install built-in presets
	for _, p := range DefaultPresets() {
		copyPreset := p
		copyPreset.IsBuiltin = true
		s.combos[copyPreset.ID] = &copyPreset
	}

	// 2. Load global combos (~/.mncode/combos/*.json)
	s.loadDirLocked(s.globalDir)

	// 3. Load workspace combos (.mncode/combos/*.json)
	if s.workspaceDir != "" {
		localDir := filepath.Join(s.workspaceDir, ".mncode", "combos")
		s.loadDirLocked(localDir)
	}

	return nil
}

func (s *Store) loadDirLocked(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var c Combo
		if err := json.Unmarshal(data, &c); err == nil && c.ID != "" {
			c.IsBuiltin = false
			s.combos[c.ID] = &c
		}
	}
}

// Get finds a combo by ID or slug.
func (s *Store) Get(id string) (*Combo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.combos[strings.ToLower(strings.TrimSpace(id))]
	return c, ok
}

// List returns all registered combos sorted with built-ins first.
func (s *Store) List() []Combo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var builtins []Combo
	var customs []Combo
	for _, c := range s.combos {
		if c.IsBuiltin {
			builtins = append(builtins, *c)
		} else {
			customs = append(customs, *c)
		}
	}
	return append(builtins, customs...)
}

// Save writes a user combo to the global storage path atomically.
func (s *Store) Save(c Combo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(c.ID) == "" {
		c.ID = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(c.Name), " ", "-"))
	}
	if c.ID == "" {
		return fmt.Errorf("combo ID or Name is required")
	}
	if len(c.Members) == 0 {
		return fmt.Errorf("combo must contain at least one member")
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	c.UpdatedAt = time.Now()
	c.IsBuiltin = false

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal combo: %w", err)
	}

	targetPath := filepath.Join(s.globalDir, fmt.Sprintf("%s.json", c.ID))
	tmpPath := fmt.Sprintf("%s.tmp-%d", targetPath, time.Now().UnixNano())

	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("write temp combo: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit combo file: %w", err)
	}

	s.combos[c.ID] = &c
	return nil
}

// Delete removes a custom combo from global storage. Built-ins cannot be deleted.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.ToLower(strings.TrimSpace(id))
	c, ok := s.combos[id]
	if !ok {
		return fmt.Errorf("combo %q not found", id)
	}
	if c.IsBuiltin {
		return fmt.Errorf("cannot delete official built-in combo %q", id)
	}

	targetPath := filepath.Join(s.globalDir, fmt.Sprintf("%s.json", id))
	_ = os.Remove(targetPath)
	delete(s.combos, id)
	return nil
}
