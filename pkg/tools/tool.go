package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Tool is the interface all agent tools must implement.
//
// Tool intentionally remains small and source-compatible. Optional metadata is
// supplied through ToolSpec rather than by extending this interface.
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// ToolScope describes the boundary in which a tool is allowed to operate.
// Scope is metadata for callers that enforce workspace/session policy.
type ToolScope string

const (
	ScopeGlobal    ToolScope = "global"
	ScopeWorkspace ToolScope = "workspace"
	ScopeSession   ToolScope = "session"
	ScopeRun       ToolScope = "run"
)

// AsyncHandler is an optional asynchronous execution implementation.
type AsyncHandler func(context.Context, map[string]interface{}) (string, error)

// AsyncMetadata describes asynchronous support without requiring every Tool
// implementation to grow a second execution method.
type AsyncMetadata struct {
	Supported bool
	Handler   AsyncHandler
	Timeout   time.Duration
}

// ToolSpec layers policy and execution metadata over a source-compatible Tool.
//
// Availability and required environment checks are evaluated by Registry
// model-facing methods and before Registry.Execute. Available is provided for
// simple checks that do not need context; Availability is preferred when a
// caller needs cancellation or request-scoped state.
type ToolSpec struct {
	Tool          Tool
	Toolset       string
	Toolsets      []string
	Available     func() bool
	Availability  func(context.Context) bool
	RequiredEnv   []string
	MaxResultSize int
	Scope         ToolScope
	Async         AsyncMetadata
	AsyncCapable  bool
	SupportsAsync bool
}

func cloneToolSpec(spec ToolSpec) ToolSpec {
	spec.RequiredEnv = append([]string(nil), spec.RequiredEnv...)
	spec.Toolsets = append([]string(nil), spec.Toolsets...)
	return spec
}

// Name delegates to the wrapped tool, allowing a ToolSpec to be used where a
// Tool is expected.
func (s ToolSpec) Name() string {
	if s.Tool == nil {
		return ""
	}
	return s.Tool.Name()
}

func (s ToolSpec) Description() string {
	if s.Tool == nil {
		return ""
	}
	return s.Tool.Description()
}

func (s ToolSpec) Schema() map[string]interface{} {
	if s.Tool == nil {
		return nil
	}
	return s.Tool.Schema()
}

// IsAvailable reports whether this specification can be exposed or executed.
func (s ToolSpec) IsAvailable(ctx context.Context) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	for _, name := range s.RequiredEnv {
		if strings.TrimSpace(name) == "" {
			continue
		}
		if value, ok := os.LookupEnv(name); !ok || strings.TrimSpace(value) == "" {
			return false
		}
	}
	if s.Availability != nil && !s.Availability(ctx) {
		return false
	}
	if s.Available != nil && !s.Available() {
		return false
	}
	return s.Tool != nil
}

// AsyncSupported reports whether the spec advertises async execution.
func (s ToolSpec) AsyncSupported() bool {
	return s.Async.Supported || s.Async.Handler != nil || s.AsyncCapable || s.SupportsAsync
}

// Execute applies availability and result-size policy to the wrapped Tool.
func (s ToolSpec) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if s.Tool == nil {
		return "", fmt.Errorf("tool spec has no tool")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !s.IsAvailable(ctx) {
		return "", fmt.Errorf("tool '%s' is unavailable", s.Name())
	}
	result, err := s.Tool.Execute(ctx, args)
	if err != nil {
		return "", err
	}
	return limitToolResult(result, s.MaxResultSize), nil
}

// ExecuteAsync uses the optional asynchronous handler and applies the same
// availability and result-size policy as synchronous execution.
func (s ToolSpec) ExecuteAsync(ctx context.Context, args map[string]interface{}) (string, error) {
	if !s.AsyncSupported() {
		return "", fmt.Errorf("tool '%s' does not support asynchronous execution", s.Name())
	}
	if !s.IsAvailable(ctx) {
		return "", fmt.Errorf("tool '%s' is unavailable", s.Name())
	}
	if ctx == nil {
		ctx = context.Background()
	}
	execCtx := ctx
	cancel := func() {}
	if s.Async.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, s.Async.Timeout)
	}
	defer cancel()
	if s.Async.Handler == nil {
		return s.Execute(execCtx, args)
	}
	result, err := s.Async.Handler(execCtx, args)
	if err != nil {
		return "", err
	}
	return limitToolResult(result, s.MaxResultSize), nil
}

func limitToolResult(result string, maxBytes int) string {
	if maxBytes <= 0 || len(result) <= maxBytes {
		return result
	}
	result = result[:maxBytes]
	for len(result) > 0 && !utf8.ValidString(result) {
		_, size := utf8.DecodeLastRuneInString(result)
		if size == 0 {
			break
		}
		result = result[:len(result)-size]
	}
	return result
}

// ToolDefinition is the deterministic, model-facing representation of a tool.
// It mirrors provider tool definitions without coupling this package to a
// provider implementation.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type registrationOwner uint8

const (
	registrationNative registrationOwner = iota
	registrationMCP
)

type registryEntry struct {
	spec  ToolSpec
	owner registrationOwner
}

// Registry manages available tools and their optional metadata.
type Registry struct {
	mu      sync.RWMutex
	tools   map[string]Tool
	specs   map[string]ToolSpec
	entries map[string]registryEntry
}

// NewRegistry creates a new empty Tool Registry.
func NewRegistry() *Registry {
	return &Registry{
		tools:   make(map[string]Tool),
		specs:   make(map[string]ToolSpec),
		entries: make(map[string]registryEntry),
	}
}

// Close releases resources owned by registered tools that implement io.Closer.
// It is idempotent for the built-in kernel and debugger tools.
func (r *Registry) Close() error {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	specs := make([]ToolSpec, 0, len(r.specs))
	for _, spec := range r.specs {
		specs = append(specs, spec)
	}
	r.mu.RUnlock()
	var firstErr error
	for _, spec := range specs {
		closer, ok := spec.Tool.(io.Closer)
		if !ok {
			continue
		}
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Register adds a tool to the registry, preserving the historical replacement
// behavior for callers that register a tool with an existing name.
func (r *Registry) Register(tool Tool) {
	if r == nil || tool == nil || strings.TrimSpace(tool.Name()) == "" {
		return
	}
	r.registerSpec(ToolSpec{Tool: tool}, registrationNative, true)
}

// RegisterSpec adds a metadata-rich tool specification. Like Register, it
// replaces an existing registration with the same name.
func (r *Registry) RegisterSpec(spec ToolSpec) {
	if r == nil || spec.Tool == nil || strings.TrimSpace(spec.Name()) == "" {
		return
	}
	r.registerSpec(spec, registrationNative, true)
}

// RegisterToolSpec is a descriptive alias for RegisterSpec.
func (r *Registry) RegisterToolSpec(spec ToolSpec) {
	r.RegisterSpec(spec)
}

// registerMCPSpec adds an MCP entry only when no existing tool owns the name.
// MCP refreshes use this method so a remote server can never replace a native
// tool (or an MCP tool from an earlier server in the same refresh).
func (r *Registry) registerMCPSpec(spec ToolSpec) bool {
	if r == nil || spec.Tool == nil || strings.TrimSpace(spec.Name()) == "" {
		return false
	}
	return r.registerSpec(spec, registrationMCP, false)
}
func (r *Registry) registerSpec(spec ToolSpec, owner registrationOwner, replace bool) bool {
	name := spec.Name()
	r.mu.Lock()
	defer r.mu.Unlock()
	if !replace {
		if _, exists := r.entries[name]; exists {
			return false
		}
	}
	if r.tools == nil {
		r.tools = make(map[string]Tool)
	}
	if r.specs == nil {
		r.specs = make(map[string]ToolSpec)
	}
	if r.entries == nil {
		r.entries = make(map[string]registryEntry)
	}
	spec = cloneToolSpec(spec)
	r.entries[name] = registryEntry{spec: spec, owner: owner}
	r.tools[name] = spec.Tool
	r.specs[name] = spec
	return true
}

// removeOwner removes only entries installed by the given registration owner.
func (r *Registry) removeOwner(owner registrationOwner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, entry := range r.entries {
		if entry.owner != owner {
			continue
		}
		delete(r.entries, name)
		delete(r.tools, name)
		delete(r.specs, name)
	}
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// GetSpec retrieves a tool's metadata-rich specification.
func (r *Registry) GetSpec(name string) (ToolSpec, bool) {
	if r == nil {
		return ToolSpec{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	spec, ok := r.specs[name]
	return cloneToolSpec(spec), ok
}

// Unregister removes a tool from the registry. It returns true when a tool
// was present and removed.
func (r *Registry) Unregister(name string) bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[name]; !ok {
		return false
	}
	delete(r.entries, name)
	delete(r.tools, name)
	delete(r.specs, name)
	return true
}

// All returns all registered tools in deterministic name order.
func (r *Registry) All() []Tool {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	list := make([]Tool, 0, len(names))
	for _, name := range names {
		list = append(list, r.tools[name])
	}
	r.mu.RUnlock()
	return list
}

// Specs returns registered specifications in deterministic name order. When a
// context is supplied, unavailable specs are omitted.
func (r *Registry) Specs(contexts ...context.Context) []ToolSpec {
	if r == nil {
		return nil
	}
	ctx := context.Background()
	if len(contexts) > 0 && contexts[0] != nil {
		ctx = contexts[0]
	}
	r.mu.RLock()
	names := make([]string, 0, len(r.specs))
	for name := range r.specs {
		names = append(names, name)
	}
	sort.Strings(names)
	allSpecs := make([]ToolSpec, 0, len(names))
	for _, name := range names {
		allSpecs = append(allSpecs, cloneToolSpec(r.specs[name]))
	}
	r.mu.RUnlock()

	specs := make([]ToolSpec, 0, len(allSpecs))
	for _, spec := range allSpecs {
		if spec.IsAvailable(ctx) {
			specs = append(specs, spec)
		}
	}
	return specs
}

// Definitions returns available model-facing tool definitions sorted by name.
// A variadic context keeps the no-argument form convenient for existing
// callers while allowing request-scoped availability checks.
func (r *Registry) Definitions(contexts ...context.Context) []ToolDefinition {
	specs := r.Specs(contexts...)
	definitions := make([]ToolDefinition, 0, len(specs))
	for _, spec := range specs {
		definitions = append(definitions, ToolDefinition{
			Name:        spec.Name(),
			Description: spec.Description(),
			InputSchema: spec.Schema(),
		})
	}
	return definitions
}

// DefinitionsForToolset returns available definitions belonging to toolset.
func (r *Registry) DefinitionsForToolset(toolset string, contexts ...context.Context) []ToolDefinition {
	toolset = strings.TrimSpace(toolset)
	if toolset == "" {
		return r.Definitions(contexts...)
	}
	specs := r.Specs(contexts...)
	definitions := make([]ToolDefinition, 0, len(specs))
	for _, spec := range specs {
		if spec.Toolset != toolset && !containsString(spec.Toolsets, toolset) {
			continue
		}
		definitions = append(definitions, ToolDefinition{
			Name:        spec.Name(),
			Description: spec.Description(),
			InputSchema: spec.Schema(),
		})
	}
	return definitions
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// Execute calls a registered tool by name, applying metadata policy.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	spec, ok := r.GetSpec(name)
	if !ok {
		return "", fmt.Errorf("tool '%s' not found", name)
	}
	return spec.Execute(ctx, args)
}
