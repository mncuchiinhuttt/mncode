package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Tracker struct {
	mu          sync.RWMutex
	filePath    string
	store       UsageStore
	lastSaveErr error
}

func NewTracker() *Tracker {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".mncode")
	_ = os.MkdirAll(dir, 0700)
	return NewTrackerAt(filepath.Join(dir, "usage.json"))
}

// NewTrackerAt opens a tracker at an explicit path. It is useful for callers
// with an isolated profile and for deterministic tests.
func NewTrackerAt(filePath string) *Tracker {
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
	_ = t.load()
	return t
}

func (t *Tracker) load() error {
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.lastSaveErr = fmt.Errorf("read usage store: %w", err)
		return t.lastSaveErr
	}
	if err := json.Unmarshal(data, &t.store); err != nil {
		t.lastSaveErr = fmt.Errorf("decode usage store: %w", err)
		return t.lastSaveErr
	}
	t.ensureMaps()
	return nil
}

func (t *Tracker) ensureMaps() {
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
	if t.store.History == nil {
		t.store.History = make([]TokenRecord, 0)
	}
}

func (t *Tracker) save() error {
	data, err := json.MarshalIndent(t.store, "", "  ")
	if err != nil {
		t.lastSaveErr = fmt.Errorf("encode usage store: %w", err)
		return t.lastSaveErr
	}
	if err := os.WriteFile(t.filePath, data, 0600); err != nil {
		t.lastSaveErr = fmt.Errorf("write usage store: %w", err)
		return t.lastSaveErr
	}
	t.lastSaveErr = nil
	return nil
}

// LastError reports the most recent persistence error, if any.
func (t *Tracker) LastError() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastSaveErr
}

// Records returns a snapshot of the records currently retained by the tracker.
func (t *Tracker) Records() []TokenRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	records := make([]TokenRecord, len(t.store.History))
	copy(records, t.store.History)
	return records
}

// Record registers a single LLM request's token usage.
func (t *Tracker) Record(model, accountID string, inputTokens, outputTokens int) {
	_ = t.RecordWithThinking(model, accountID, inputTokens, outputTokens, 0)
}

// RecordWithThinking registers usage while preserving provider-reported
// thinking/reasoning tokens separately from visible output tokens.
func (t *Tracker) RecordWithThinking(model, accountID string, inputTokens, outputTokens, thinkingTokens int) error {
	if inputTokens < 0 {
		inputTokens = 0
	}
	if outputTokens < 0 {
		outputTokens = 0
	}
	if thinkingTokens < 0 {
		thinkingTokens = 0
	}
	in := int64(inputTokens)
	out := int64(outputTokens)
	thinking := int64(thinkingTokens)
	total := in + out + thinking
	if total == 0 {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.ensureMaps()

	now := time.Now().UTC()
	dateKey := now.Format("2006-01-02")
	monthKey := now.Format("2006-01")

	add := func(sum *UsageSummary) {
		sum.InputTokens += in
		sum.OutputTokens += out
		sum.ThinkingTokens += thinking
		sum.TotalTokens += total
		sum.Requests++
	}

	if t.store.Daily[dateKey] == nil {
		t.store.Daily[dateKey] = &UsageSummary{}
	}
	add(t.store.Daily[dateKey])

	if t.store.Monthly[monthKey] == nil {
		t.store.Monthly[monthKey] = &UsageSummary{}
	}
	add(t.store.Monthly[monthKey])

	if model != "" {
		if t.store.ByModel[model] == nil {
			t.store.ByModel[model] = &UsageSummary{}
		}
		add(t.store.ByModel[model])
	}

	add(t.store.Lifetime)

	t.store.History = append(t.store.History, TokenRecord{
		Timestamp:      now,
		Date:           dateKey,
		Month:          monthKey,
		Model:          model,
		AccountID:      accountID,
		InputTokens:    in,
		OutputTokens:   out,
		ThinkingTokens: thinking,
		TotalTokens:    total,
	})
	if len(t.store.History) > 100 {
		t.store.History = t.store.History[len(t.store.History)-100:]
	}

	return t.save()
}

func (t *Tracker) GetToday() UsageSummary {
	t.mu.RLock()
	defer t.mu.RUnlock()
	dateKey := time.Now().UTC().Format("2006-01-02")
	if sum, ok := t.store.Daily[dateKey]; ok && sum != nil {
		return *sum
	}
	return UsageSummary{}
}

func (t *Tracker) GetMonth() UsageSummary {
	t.mu.RLock()
	defer t.mu.RUnlock()
	monthKey := time.Now().UTC().Format("2006-01")
	if sum, ok := t.store.Monthly[monthKey]; ok && sum != nil {
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
	for model, summary := range t.store.ByModel {
		if summary != nil {
			res[model] = *summary
		}
	}
	return res
}
