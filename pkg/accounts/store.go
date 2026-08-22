package accounts

import (
	"encoding/json"
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

	s.Accounts = loaded
	return nil
}

// Save writes accounts to disk safely
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.Accounts, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0600)
}

// AddOrUpdate saves an account into the store
func (s *Store) AddOrUpdate(acc *Account) error {
	s.mu.Lock()
	if acc.CreatedAt.IsZero() {
		acc.CreatedAt = time.Now()
	}
	s.Accounts[acc.ID] = acc
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

// ListByProvider returns all accounts for a specific provider sorted by ID
func (s *Store) ListByProvider(provider AccountProvider) []*Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*Account
	for _, acc := range s.Accounts {
		if acc.Provider == provider {
			list = append(list, acc)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list
}

// GetActiveAccount returns the explicitly selected active account for a provider
func (s *Store) GetActiveAccount(provider AccountProvider) *Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var fallback *Account
	for _, acc := range s.Accounts {
		if acc.Provider == provider {
			if acc.IsActive {
				return acc
			}
			if fallback == nil {
				fallback = acc
			}
		}
	}
	return fallback
}
