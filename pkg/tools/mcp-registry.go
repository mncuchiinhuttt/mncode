package tools

import (
	"context"
	"sort"
	"strings"

	"mncode/pkg/mcp"
)

// RegisterMCPTools registers all tools exposed by connected MCP servers.
//
// Registration is a replace-only refresh for MCP-owned entries. Native tools
// are never overridden by a remote server, and duplicate server/tool names are
// resolved deterministically by sorted server and tool names.
func RegisterMCPTools(registry *Registry, mgr *mcp.Manager, ctx context.Context) int {
	if registry == nil || mgr == nil {
		return 0
	}
	if ctx == nil {
		ctx = context.Background()
	}

	registry.removeOwner(registrationMCP)

	statuses := mgr.GetStatus(ctx)
	sort.SliceStable(statuses, func(i, j int) bool {
		return statuses[i].Name < statuses[j].Name
	})

	count := 0
	for _, st := range statuses {
		if !st.Connected {
			continue
		}
		client := mgr.GetClient(st.Name)
		if client == nil {
			continue
		}
		tools := append([]mcp.MCPToolInfo(nil), st.Tools...)
		sort.SliceStable(tools, func(i, j int) bool {
			return tools[i].Name < tools[j].Name
		})
		for _, tInfo := range tools {
			if strings.TrimSpace(tInfo.Name) == "" {
				continue
			}
			wrapper := &MCPToolWrapper{
				Client: client,
				Info:   tInfo,
			}
			spec := ToolSpec{
				Tool:    wrapper,
				Toolset: "mcp:" + st.Name,
				Scope:   ScopeSession,
			}
			if registry.registerMCPSpec(spec) {
				count++
			}
		}
	}
	return count
}
