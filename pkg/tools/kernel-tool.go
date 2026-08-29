package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// KernelTool runs persistent Python or Node namespaces. Each session has an
// independent gate so a slow calculation cannot block other sessions.
type KernelTool struct {
	BaseDir string
	mu      sync.Mutex
	kernels map[string]*kernelEntry
}

type kernelEntry struct {
	gate    chan struct{}
	process *kernelProcess
}

// Name returns the model-facing tool name.
func (k *KernelTool) Name() string { return "persistent_kernel" }

// Description explains the persistent interpreter contract.
func (k *KernelTool) Description() string {
	return "Execute Python or Node JavaScript in a persistent session. Variables survive across calls and output is bounded."
}

// Schema returns the kernel execution schema.
func (k *KernelTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"language":   map[string]interface{}{"type": "string", "enum": []string{"python", "node"}, "description": "Persistent interpreter; defaults to python."},
			"session_id": map[string]interface{}{"type": "string", "description": "Namespace identifier; defaults to default."},
			"code":       map[string]interface{}{"type": "string", "description": "Code to execute in the persistent namespace."},
			"reset":      map[string]interface{}{"type": "boolean", "description": "Terminate the namespace before executing code."},
		},
		"required": []string{"code"},
	}
}

// Execute runs code and returns bounded stdout, stderr, and the last result.
func (k *KernelTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	code, _ := args["code"].(string)
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("code is required")
	}
	language, _ := args["language"].(string)
	language = normalizeKernelLanguage(language)
	sessionID, _ := args["session_id"].(string)
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "default"
	}
	if len(sessionID) > 128 || strings.ContainsAny(sessionID, "/\\\x00") {
		return "", fmt.Errorf("session_id must be <=128 safe characters")
	}
	reset, _ := args["reset"].(bool)

	entry := k.entry(sessionID)
	if err := entry.acquire(ctx); err != nil {
		return "", err
	}
	defer entry.release()
	if reset && entry.process != nil {
		entry.process.stop()
		entry.process = nil
	}
	if entry.process == nil {
		process, err := startKernel(language, k.BaseDir)
		if err != nil {
			return "", err
		}
		entry.process = process
	}
	if entry.process.language != language {
		return "", fmt.Errorf("kernel session %q is already using %s; set reset=true before switching to %s", sessionID, entry.process.language, language)
	}
	result, err := entry.process.execute(ctx, code)
	if err != nil {
		entry.process.stop()
		entry.process = nil
		return "", err
	}
	return formatKernelResponse(result), nil
}

// Close terminates all persistent interpreter processes owned by the tool.
func (k *KernelTool) Close() error {
	k.mu.Lock()
	entries := make([]*kernelEntry, 0, len(k.kernels))
	for _, entry := range k.kernels {
		entries = append(entries, entry)
	}
	k.mu.Unlock()
	for _, entry := range entries {
		if err := entry.acquire(context.Background()); err != nil {
			continue
		}
		if entry.process != nil {
			entry.process.stop()
			entry.process = nil
		}
		entry.release()
	}
	return nil
}

func (k *KernelTool) entry(sessionID string) *kernelEntry {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.kernels == nil {
		k.kernels = make(map[string]*kernelEntry)
	}
	entry := k.kernels[sessionID]
	if entry == nil {
		entry = &kernelEntry{gate: make(chan struct{}, 1)}
		entry.gate <- struct{}{}
		k.kernels[sessionID] = entry
	}
	return entry
}

func (e *kernelEntry) acquire(ctx context.Context) error {
	select {
	case <-e.gate:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *kernelEntry) release() { e.gate <- struct{}{} }

func normalizeKernelLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "node", "nodejs", "javascript", "js":
		return "node"
	default:
		return "python"
	}
}
