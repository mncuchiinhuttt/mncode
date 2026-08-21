package tools

// DefaultRegistry creates a tool registry with all standard tools initialized
func DefaultRegistry(workspaceDir string, autoApprove bool) *Registry {
	reg := NewRegistry()

	reg.Register(&BashTool{DefaultCwd: workspaceDir})
	reg.Register(&ViewTool{BaseDir: workspaceDir})
	reg.Register(&EditTool{BaseDir: workspaceDir})
	reg.Register(&WriteTool{BaseDir: workspaceDir})
	reg.Register(&GrepTool{BaseDir: workspaceDir})
	reg.Register(&FindTool{BaseDir: workspaceDir})
	reg.Register(&ListTool{BaseDir: workspaceDir})
	reg.Register(&WebTool{})
	reg.Register(&SearchWebTool{})
	reg.Register(&ViewImageTool{BaseDir: workspaceDir})
	reg.Register(&AskTool{AutoApprove: autoApprove})

	return reg
}
