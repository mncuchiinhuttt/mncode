package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// HierarchicalStore manages Global Master Memory and Shared Workspace Memory concurrently.
type HierarchicalStore struct {
	mu             sync.RWMutex
	workspaceDir   string
	globalPath     string
	workspacePath  string
	globalItems    map[string]*MemoryItem
	workspaceItems map[string]*MemoryItem
}

// NewHierarchicalStore initializes 2-tier memory for the given workspace.
func NewHierarchicalStore(workspaceDir string) (*HierarchicalStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	globalDir := filepath.Join(home, ".mncode", "memory")
	_ = os.MkdirAll(globalDir, 0700)
	globalPath := filepath.Join(globalDir, "global.json")

	workspacePath := ""
	if workspaceDir != "" {
		wsDir := filepath.Join(workspaceDir, ".mncode", "memory")
		_ = os.MkdirAll(wsDir, 0700)
		workspacePath = filepath.Join(wsDir, "workspace.json")
	}

	return NewHierarchicalStoreWithPaths(globalPath, workspacePath), nil
}

// NewHierarchicalStoreWithPaths initializes store with custom explicit file paths (useful for isolated tests).
func NewHierarchicalStoreWithPaths(globalPath, workspacePath string) *HierarchicalStore {
	s := &HierarchicalStore{
		globalPath:     globalPath,
		workspacePath:  workspacePath,
		globalItems:    make(map[string]*MemoryItem),
		workspaceItems: make(map[string]*MemoryItem),
	}
	_ = s.Load()
	return s
}

// Load reads both global and workspace memories from disk.
func (s *HierarchicalStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.globalItems = s.loadFileLocked(s.globalPath, TierGlobal)
	if s.workspacePath != "" {
		s.workspaceItems = s.loadFileLocked(s.workspacePath, TierWorkspace)
	}
	return nil
}

func (s *HierarchicalStore) loadFileLocked(path string, tier MemoryTier) map[string]*MemoryItem {
	items := make(map[string]*MemoryItem)
	data, err := os.ReadFile(path)
	if err != nil {
		return items
	}
	var list []MemoryItem
	if err := json.Unmarshal(data, &list); err == nil {
		for i := range list {
			item := list[i]
			item.Tier = tier
			items[item.ID] = &item
		}
	}
	return items
}

// Save persists a memory item to its designated tier atomically.
func (s *HierarchicalStore) Save(item MemoryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		hash := sha256.Sum256([]byte(item.Topic + item.Summary + fmt.Sprintf("-%d", time.Now().UnixNano())))
		item.ID = "mem-" + hex.EncodeToString(hash[:])[:8]
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = time.Now()
	if item.Confidence <= 0 {
		item.Confidence = 5
	}

	targetMap := s.workspaceItems
	targetPath := s.workspacePath
	if item.Tier == TierGlobal || s.workspacePath == "" {
		item.Tier = TierGlobal
		targetMap = s.globalItems
		targetPath = s.globalPath
	}

	targetMap[item.ID] = &item
	return s.saveFileLocked(targetPath, targetMap)
}

func (s *HierarchicalStore) saveFileLocked(path string, items map[string]*MemoryItem) error {
	if path == "" {
		return nil
	}
	var list []MemoryItem
	for _, it := range items {
		list = append(list, *it)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal memory: %w", err)
	}

	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0700)
	tmpPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UnixNano())

	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("write temp memory: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit memory file: %w", err)
	}
	return nil
}

// ListAll returns all memories across both workspace and global tiers.
func (s *HierarchicalStore) ListAll() []MemoryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []MemoryItem
	for _, it := range s.workspaceItems {
		all = append(all, *it)
	}
	for _, it := range s.globalItems {
		all = append(all, *it)
	}
	return all
}

// Delete removes a memory item by ID from whichever tier contains it.
func (s *HierarchicalStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if _, ok := s.workspaceItems[id]; ok {
		delete(s.workspaceItems, id)
		return s.saveFileLocked(s.workspacePath, s.workspaceItems)
	}
	if _, ok := s.globalItems[id]; ok {
		delete(s.globalItems, id)
		return s.saveFileLocked(s.globalPath, s.globalItems)
	}
	return fmt.Errorf("memory item %q not found", id)
}
