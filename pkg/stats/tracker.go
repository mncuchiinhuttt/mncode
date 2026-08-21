package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Tracker struct {
	mu       sync.RWMutex
	filePath string
	store    UsageStore
}

func NewTracker() *Tracker {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".mncode")
	_ = os.MkdirAll(dir, 0755)
	filePath := filepath.Join(dir, "usage.json")

	t := &Tracker{
		filePath: filePath,
		store: UsageStore{
			Daily:    make(map[string]*UsageSummary),
			Monthly:  make(map[string]*UsageSummary),
			ByModel:  make(map[string]*UsageSummary),
			Lifetime: &UsageSummary{},
			History:  make([]TokenRecord, 0),
		},
	}
	t.load()
	return t
}

func (t *Tracker) load() {
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &t.store)
	if t.store.Daily == nil {
		t.store.Daily = make(map[string]*UsageSummary)
	}
	if t.store.Monthly == nil {
		t.store.Monthly = make(map[string]*UsageSummary)
	}
	if t.store.ByModel == nil {
		t.store.ByModel = make(map[string]*UsageSummary)
	}
	if t.store.Lifetime == nil {
		t.store.Lifetime = &UsageSummary{}
	}
}

func (t *Tracker) save() {
	data, err := json.MarshalIndent(t.store, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(t.filePath, data, 0644)
}

// Record registers a single LLM request's token usage
func (t *Tracker) Record(model, accountID string, inputTokens, outputTokens int) {
	if inputTokens < 0 {
		inputTokens = 0
	}
	if outputTokens < 0 {
		outputTokens = 0
	}
	total := int64(inputTokens + outputTokens)
	if total == 0 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	dateKey := now.Format("2006-01-02")
	monthKey := now.Format("2006-01")

	in := int64(inputTokens)
	out := int64(outputTokens)

	// 1. Update Daily
	if _, ok := t.store.Daily[dateKey]; !ok {
		t.store.Daily[dateKey] = &UsageSummary{}
	}
	t.store.Daily[dateKey].InputTokens += in
	t.store.Daily[dateKey].OutputTokens += out
	t.store.Daily[dateKey].TotalTokens += total
	t.store.Daily[dateKey].Requests++

	// 2. Update Monthly
	if _, ok := t.store.Monthly[monthKey]; !ok {
		t.store.Monthly[monthKey] = &UsageSummary{}
	}
	t.store.Monthly[monthKey].InputTokens += in
	t.store.Monthly[monthKey].OutputTokens += out
	t.store.Monthly[monthKey].TotalTokens += total
	t.store.Monthly[monthKey].Requests++

	// 3. Update Model
	if model != "" {
		if _, ok := t.store.ByModel[model]; !ok {
			t.store.ByModel[model] = &UsageSummary{}
		}
		t.store.ByModel[model].InputTokens += in
		t.store.ByModel[model].OutputTokens += out
		t.store.ByModel[model].TotalTokens += total
		t.store.ByModel[model].Requests++
	}

	// 4. Update Lifetime
	t.store.Lifetime.InputTokens += in
	t.store.Lifetime.OutputTokens += out
	t.store.Lifetime.TotalTokens += total
	t.store.Lifetime.Requests++

	// 5. Append recent history (keep up to 100 entries)
	rec := TokenRecord{
		Timestamp:    now,
		Date:         dateKey,
		Month:        monthKey,
		Model:        model,
		AccountID:    accountID,
		InputTokens:  in,
		OutputTokens: out,
		TotalTokens:  total,
	}
	t.store.History = append(t.store.History, rec)
	if len(t.store.History) > 100 {
		t.store.History = t.store.History[len(t.store.History)-100:]
	}

	t.save()
}

func (t *Tracker) GetToday() UsageSummary {
	t.mu.RLock()
	defer t.mu.RUnlock()
	dateKey := time.Now().Format("2006-01-02")
	if sum, ok := t.store.Daily[dateKey]; ok {
		return *sum
	}
	return UsageSummary{}
}

func (t *Tracker) GetMonth() UsageSummary {
	t.mu.RLock()
	defer t.mu.RUnlock()
	monthKey := time.Now().Format("2006-01")
	if sum, ok := t.store.Monthly[monthKey]; ok {
		return *sum
	}
	return UsageSummary{}
}

func (t *Tracker) GetLifetime() UsageSummary {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.store.Lifetime != nil {
		return *t.store.Lifetime
	}
	return UsageSummary{}
}

func (t *Tracker) GetByModel() map[string]UsageSummary {
	t.mu.RLock()
	defer t.mu.RUnlock()
	res := make(map[string]UsageSummary)
	for k, v := range t.store.ByModel {
		res[k] = *v
	}
	return res
}
