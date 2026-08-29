package tools

import (
	"mncode/pkg/accounts"
	"mncode/pkg/browserctl"
	"mncode/pkg/config"
)

// DefaultRegistry creates a tool registry with all standard tools initialized.
// An optional account store lets search_web reuse the active Antigravity token.
func DefaultRegistry(workspaceDir string, autoApprove bool, cfg *config.Config, accountStores ...*accounts.Store) *Registry {
	reg := NewRegistry()
	register := func(tool Tool, toolset string, scope ToolScope) {
		reg.RegisterSpec(ToolSpec{Tool: tool, Toolset: toolset, Scope: scope})
	}

	register(&BashTool{DefaultCwd: workspaceDir}, "workspace", ScopeWorkspace)
	register(&ViewTool{BaseDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&EditTool{BaseDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&WriteTool{BaseDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&GrepTool{BaseDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&FindTool{BaseDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&ListTool{BaseDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&WebTool{}, "network", ScopeSession)
	searchTool := &SearchWebTool{Config: cfg}
	if len(accountStores) > 0 {
		searchTool.Accounts = accountStores[0]
	}
	register(searchTool, "network", ScopeSession)
	register(&ViewImageTool{BaseDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&AskTool{AutoApprove: autoApprove}, "interactive", ScopeSession)
	register(&SymbolTool{WorkspaceDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&LSPTool{BaseDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&KernelTool{BaseDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&DAPTool{WorkspaceDir: workspaceDir}, "workspace", ScopeWorkspace)
	register(&ServiceHubTool{DefaultCwd: workspaceDir}, "workspace", ScopeWorkspace)
	register(&ASTEditTool{BaseDir: workspaceDir}, "workspace", ScopeWorkspace)

	browser := &BrowserTool{
		Enabled: func() bool {
			return cfg != nil && cfg.GetSetting("browser_control_enabled", "false") == "true"
		},
		SessionFactory: func() *browserctl.Session {
			opts := browserctl.Options{UserDataDir: browserctl.DefaultUserDataDir()}
			if cfg != nil {
				opts.IgnoreCertErrors = cfg.GetSetting("browser_ignore_cert_errors", "false") == "true"
				opts.Headless = cfg.GetSetting("browser_headless", "false") == "true"
			}
			return browserctl.Shared(opts)
		},
	}
	reg.RegisterSpec(ToolSpec{
		Tool:      browser,
		Toolset:   "browser",
		Scope:     ScopeSession,
		Available: browser.Enabled,
	})

	return reg
}
