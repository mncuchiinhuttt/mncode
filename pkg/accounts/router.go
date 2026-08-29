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

// GetNextAccount selects the active account if available, or rotates among healthy accounts
func (r *Router) GetNextAccount(provider AccountProvider) (*Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	all := r.store.ListByProvider(provider)
	if len(all) == 0 {
		return nil, fmt.Errorf("no accounts found for provider '%s'. Use /account add to add one", provider)
	}

	// 1. Prefer explicitly marked Active account if available
	for _, acc := range all {
		if acc.IsActive && acc.IsAvailable() {
			acc.UsageCount++
			_ = r.store.AddOrUpdate(acc)
			return acc, nil
		}
	}

	// 2. Otherwise fallback to available healthy accounts in rotation
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

// ReportFailure marks an account on cooldown for retryable provider failures.
func (r *Router) ReportFailure(accountID string, statusCode int, errMsg string) {
	if statusCode != 0 && statusCode != 401 && statusCode != 403 && statusCode != 408 && statusCode != 425 && statusCode != 429 && statusCode < 500 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		return
	}
	acc, ok := r.store.Get(accountID)
	if !ok {
		return
	}

	if statusCode == 429 {
		acc.MarkCooldown(2*time.Minute, fmt.Sprintf("Rate limited (429): %s", errMsg))
	} else if statusCode == 401 || statusCode == 403 {
		acc.MarkCooldown(10*time.Minute, fmt.Sprintf("Auth failed (%d): %s", statusCode, errMsg))
	} else {
		acc.MarkCooldown(30*time.Second, errMsg)
	}
	_ = r.store.AddOrUpdate(acc)
}

// ReportSuccess resets a failed account's cooldown and last error after a
// request succeeds. A refreshed account must be immediately eligible again.
func (r *Router) ReportSuccess(accountID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		return
	}
	if acc, ok := r.store.Get(accountID); ok {
		acc.LastError = ""
		acc.CooldownUntil = time.Time{}
		_ = r.store.AddOrUpdate(acc)
	}
}
