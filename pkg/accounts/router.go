package accounts

import (
	"fmt"
	"sync"
	"time"
)

// Router handles smart account selection and rotation like 9router
type Router struct {
	mu      sync.Mutex
	store   *Store
	indexes map[AccountProvider]int
}

// NewRouter creates a new account router
func NewRouter(store *Store) *Router {
	return &Router{
		store:   store,
		indexes: make(map[AccountProvider]int),
	}
}

// GetNextAccount selects the next healthy account using round-robin and availability checks
func (r *Router) GetNextAccount(provider AccountProvider) (*Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	all := r.store.ListByProvider(provider)
	if len(all) == 0 {
		return nil, fmt.Errorf("no accounts found for provider '%s'. Use /account add to add one", provider)
	}

	var available []*Account
	for _, acc := range all {
		if acc.IsAvailable() {
			available = append(available, acc)
		}
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("all accounts for '%s' are currently in cooldown or expired", provider)
	}

	idx := r.indexes[provider] % len(available)
	selected := available[idx]
	r.indexes[provider] = (idx + 1) % len(available)

	selected.UsageCount++
	_ = r.store.AddOrUpdate(selected)

	return selected, nil
}

// ReportFailure marks an account on cooldown upon encountering rate limits (429) or auth errors (401)
func (r *Router) ReportFailure(accountID string, statusCode int, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	acc, ok := r.store.Accounts[accountID]
	if !ok {
		return
	}

	if statusCode == 429 {
		// Rate limited: cooldown for 2 minutes
		acc.MarkCooldown(2*time.Minute, fmt.Sprintf("Rate limited (429): %s", errMsg))
	} else if statusCode == 401 || statusCode == 403 {
		// Auth error: cooldown for 10 minutes or until refreshed
		acc.MarkCooldown(10*time.Minute, fmt.Sprintf("Auth failed (%d): %s", statusCode, errMsg))
	} else {
		acc.MarkCooldown(30*time.Second, errMsg)
	}

	_ = r.store.AddOrUpdate(acc)
}

// ReportSuccess resets last error on successful usage
func (r *Router) ReportSuccess(accountID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if acc, ok := r.store.Accounts[accountID]; ok {
		acc.LastError = ""
		_ = r.store.AddOrUpdate(acc)
	}
}
