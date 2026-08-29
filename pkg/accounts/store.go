package accounts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Store manages persistence of accounts in ~/.mncode/accounts.json
type Store struct {
	mu       sync.RWMutex
	filePath string
	Accounts map[string]*Account `json:"accounts"`
}

// NewStore initializes an account store
func NewStore(customPath string) (*Store, error) {
	if customPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		customPath = filepath.Join(homeDir, ".mncode", "accounts.json")
	}

	store := &Store{
		filePath: customPath,
		Accounts: make(map[string]*Account),
	}

	_ = store.Load()
	return store, nil
}

// Load reads accounts from disk
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var loaded map[string]*Account
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	if loaded == nil {
		loaded = make(map[string]*Account)
	}
	s.Accounts = loaded
	return nil
}

// Save writes accounts to disk atomically with restrictive permissions.
func (s *Store) Save() error {
	s.mu.RLock()
	snapshot := make(map[string]*Account, len(s.Accounts))
	for id, account := range s.Accounts {
		if account == nil {
			continue
		}
		copy := *account
		snapshot[id] = &copy
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".accounts.json.tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
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
	if err := replaceExistingFile(tmpPath, s.filePath); err != nil {
		return err
	}
	return os.Chmod(s.filePath, 0600)
}

// AddOrUpdate saves an account into the store.
func (s *Store) AddOrUpdate(acc *Account) error {
	if acc == nil || acc.ID == "" {
		return fmt.Errorf("account and account ID are required")
	}
	s.mu.Lock()
	snapshot := *acc
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now()
	}
	if s.Accounts == nil {
		s.Accounts = make(map[string]*Account)
	}
	s.Accounts[snapshot.ID] = &snapshot
	s.mu.Unlock()
	return s.Save()
}

// Remove deletes an account by ID
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	delete(s.Accounts, id)
	s.mu.Unlock()

	return s.Save()
}

// ListByProvider returns account snapshots for a specific provider sorted by ID.
func (s *Store) ListByProvider(provider AccountProvider) []*Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*Account, 0)
	for _, account := range s.Accounts {
		if account.Provider != provider {
			continue
		}
		snapshot := *account
		list = append(list, &snapshot)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list
}

// Get returns a defensive snapshot for an account ID.
func (s *Store) Get(id string) (*Account, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.Accounts[id]
	if !ok || account == nil {
		return nil, false
	}
	snapshot := *account
	return &snapshot, true
}

// GetActiveAccount returns a snapshot of the explicitly selected active
// account, or the first account for the provider when none is marked active.
func (s *Store) GetActiveAccount(provider AccountProvider) *Account {
	for _, account := range s.ListByProvider(provider) {
		if account.IsActive {
			return account
		}
	}
	list := s.ListByProvider(provider)
	if len(list) > 0 {
		return list[0]
	}
	return nil
}
