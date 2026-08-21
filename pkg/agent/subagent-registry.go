package agent

import (
	"sync"
	"time"
)

type SubagentRecord struct {
	ID              string
	Name            string
	Role            string
	Prompt          string
	CurrentActivity string
	Tokens          int
	Status          string // "running", "completed", "error"
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	Turns           int
	ToolCalls       []string
	Result          string
	Logs            []string
}

type SubagentRegistry struct {
	mu      sync.RWMutex
	records []*SubagentRecord
}

func NewSubagentRegistry() *SubagentRegistry {
	return &SubagentRegistry{
		records: make([]*SubagentRecord, 0),
	}
}

func (r *SubagentRegistry) Register(id, name, role, prompt string) *SubagentRecord {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec := &SubagentRecord{
		ID:        id,
		Name:      name,
		Role:      role,
		Prompt:    prompt,
		Status:    "running",
		StartTime: time.Now(),
		ToolCalls: make([]string, 0),
		Logs:      make([]string, 0),
	}
	r.records = append(r.records, rec)
	return rec
}

func (r *SubagentRegistry) AddToolCall(id, toolName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.records {
		if rec.ID == id {
			rec.ToolCalls = append(rec.ToolCalls, toolName)
			break
		}
	}
}

func (r *SubagentRegistry) AddLog(id, logLine string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.records {
		if rec.ID == id {
			rec.Logs = append(rec.Logs, logLine)
			break
		}
	}
}

func (r *SubagentRegistry) SetActivity(id, activity string, tokens int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.records {
		if rec.ID == id {
			rec.CurrentActivity = activity
			if tokens > 0 {
				rec.Tokens = tokens
			}
			break
		}
	}
}

func (r *SubagentRegistry) Complete(id, result string, isError bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.records {
		if rec.ID == id {
			rec.EndTime = time.Now()
			rec.Duration = rec.EndTime.Sub(rec.StartTime)
			rec.Result = result
			if isError {
				rec.Status = "error"
			} else {
				rec.Status = "completed"
			}
			break
		}
	}
}

func (r *SubagentRegistry) List() []*SubagentRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*SubagentRecord, len(r.records))
	copy(list, r.records)
	return list
}

func (r *SubagentRegistry) ActiveCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, rec := range r.records {
		if rec.Status == "running" {
			count++
		}
	}
	return count
}
