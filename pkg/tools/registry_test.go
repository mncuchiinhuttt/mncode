package tools

import (
	"context"
	"testing"
)

type registryTestTool struct{ name string }

func (t registryTestTool) Name() string      { return t.name }
func (registryTestTool) Description() string { return "test tool" }
func (registryTestTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (registryTestTool) Execute(context.Context, map[string]interface{}) (string, error) {
	return "ok", nil
}

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
