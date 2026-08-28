package tools

import (
	"context"
	"strings"

	"mncode/pkg/mcp"
)

// RegisterMCPTools registers all tools exposed by connected MCP servers
func RegisterMCPTools(registry *Registry, mgr *mcp.Manager, ctx context.Context) int {
	if registry == nil || mgr == nil {
		return 0
	}

	for _, tool := range registry.All() {
		if strings.HasPrefix(tool.Name(), "mcp_") {
			registry.Unregister(tool.Name())
		}
	}

	count := 0
	statuses := mgr.GetStatus(ctx)
	for _, st := range statuses {
		if !st.Connected {
			continue
		}
		client := mgr.GetClient(st.Name)
		if client == nil {
			continue
		}
		for _, tInfo := range st.Tools {
			wrapper := &MCPToolWrapper{
				Client: client,
				Info:   tInfo,
			}
			registry.Register(wrapper)
			count++
		}
	}
	return count
}
