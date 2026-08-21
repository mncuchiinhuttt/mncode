package tools

import (
	"context"
	"fmt"
	"mncode/pkg/mcp"
)

// MCPToolWrapper exposes an MCP server tool as a native mncode agent Tool
type MCPToolWrapper struct {
	Client *mcp.Client
	Info   mcp.MCPToolInfo
}

func (m *MCPToolWrapper) Name() string {
	return fmt.Sprintf("mcp_%s_%s", m.Info.ServerName, m.Info.Name)
}

func (m *MCPToolWrapper) Description() string {
	if m.Info.Description != "" {
		return fmt.Sprintf("[%s MCP] %s", m.Info.ServerName, m.Info.Description)
	}
	return fmt.Sprintf("[%s MCP Tool: %s]", m.Info.ServerName, m.Info.Name)
}

func (m *MCPToolWrapper) Schema() map[string]interface{} {
	if m.Info.InputSchema != nil {
		return m.Info.InputSchema
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (m *MCPToolWrapper) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	return m.Client.CallTool(ctx, m.Info.Name, args)
}
