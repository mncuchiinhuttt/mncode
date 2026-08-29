package tools

import (
	"context"
	"testing"
)

type registryTestTool struct {
	name   string
	output string
}

func (t registryTestTool) Name() string      { return t.name }
func (registryTestTool) Description() string { return "test tool" }
func (registryTestTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (t registryTestTool) Execute(context.Context, map[string]interface{}) (string, error) {
	if t.output != "" {
		return t.output, nil
	}
	return "ok", nil
}

type closeableRegistryTool struct {
	closed *bool
}

func (closeableRegistryTool) Name() string        { return "closeable" }
func (closeableRegistryTool) Description() string { return "closeable test tool" }
func (closeableRegistryTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (closeableRegistryTool) Execute(context.Context, map[string]interface{}) (string, error) {
	return "ok", nil
}
func (t closeableRegistryTool) Close() error { *t.closed = true; return nil }

func TestRegistryUnregisterRemovesOnlyRequestedTool(t *testing.T) {
	registry := NewRegistry()
	registry.Register(registryTestTool{name: "mcp_demo_search"})
	registry.Register(registryTestTool{name: "view_file"})

	if !registry.Unregister("mcp_demo_search") {
		t.Fatal("expected MCP tool to be removed")
	}
	if _, ok := registry.Get("mcp_demo_search"); ok {
		t.Fatal("removed MCP tool is still registered")
	}
	if _, ok := registry.Get("view_file"); !ok {
		t.Fatal("unrelated tool was removed")
	}
	if registry.Unregister("mcp_demo_search") {
		t.Fatal("second unregister should report false")
	}
}

func TestRegistryDefinitionsAreSortedAndFilterUnavailable(t *testing.T) {
	registry := NewRegistry()
	registry.RegisterSpec(ToolSpec{Tool: registryTestTool{name: "zeta"}, Toolset: "workspace"})
	registry.RegisterSpec(ToolSpec{Tool: registryTestTool{name: "alpha"}, Toolset: "network", Availability: func(context.Context) bool { return false }})
	registry.RegisterSpec(ToolSpec{Tool: registryTestTool{name: "beta"}, Toolset: "workspace"})

	definitions := registry.Definitions(context.Background())
	if len(definitions) != 2 {
		t.Fatalf("expected 2 available definitions, got %d", len(definitions))
	}
	if definitions[0].Name != "beta" || definitions[1].Name != "zeta" {
		t.Fatalf("definitions are not sorted: %#v", definitions)
	}
	workspace := registry.DefinitionsForToolset("workspace")
	if len(workspace) != 2 || workspace[0].Name != "beta" || workspace[1].Name != "zeta" {
		t.Fatalf("unexpected workspace definitions: %#v", workspace)
	}
}

func TestRegistryExecuteAppliesResultLimitAndEnvironmentAvailability(t *testing.T) {
	registry := NewRegistry()
	registry.RegisterSpec(ToolSpec{Tool: registryTestTool{name: "limited", output: "abcdef"}, MaxResultSize: 3})
	result, err := registry.Execute(context.Background(), "limited", nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result != "abc" {
		t.Fatalf("unexpected result: %q", result)
	}

	t.Setenv("MNTestRequired", "")
	registry.RegisterSpec(ToolSpec{Tool: registryTestTool{name: "env_tool"}, RequiredEnv: []string{"MNTestRequired"}})
	if _, err := registry.Execute(context.Background(), "env_tool", nil); err == nil {
		t.Fatal("expected missing required environment to deny execution")
	}
	t.Setenv("MNTestRequired", "present")
	if _, err := registry.Execute(context.Background(), "env_tool", nil); err != nil {
		t.Fatalf("expected tool to execute with required environment: %v", err)
	}
}

func TestRegistryMCPRegistrationCannotOverrideNative(t *testing.T) {
	registry := NewRegistry()
	native := registryTestTool{name: "mcp_server_tool", output: "native"}
	registry.Register(native)
	if registry.registerMCPSpec(ToolSpec{Tool: registryTestTool{name: "mcp_server_tool", output: "remote"}, Toolset: "mcp:server"}) {
		t.Fatal("MCP registration unexpectedly replaced native tool")
	}
	result, err := registry.Execute(context.Background(), native.name, nil)
	if err != nil {
		t.Fatalf("native tool execution failed: %v", err)
	}
	if result != "native" {
		t.Fatalf("native tool was replaced: %q", result)
	}
}

func TestToolSpecAsyncHandlerUsesMetadataPolicy(t *testing.T) {
	spec := ToolSpec{Tool: registryTestTool{name: "async_tool"}, Async: AsyncMetadata{Supported: true, Handler: func(context.Context, map[string]interface{}) (string, error) { return "async-result", nil }}, MaxResultSize: 5}
	result, err := spec.ExecuteAsync(context.Background(), nil)
	if err != nil {
		t.Fatalf("async execution failed: %v", err)
	}
	if result != "async" {
		t.Fatalf("unexpected async result: %q", result)
	}
}

func TestRegistryCloseClosesOwnedTools(t *testing.T) {
	closed := false
	registry := NewRegistry()
	registry.Register(closeableRegistryTool{closed: &closed})
	if err := registry.Close(); err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatal("registry did not close an owned tool")
	}
}
