package tools

import (
	"mncode/pkg/browserctl"
	"mncode/pkg/config"
)

// DefaultRegistry creates a tool registry with all standard tools initialized
func DefaultRegistry(workspaceDir string, autoApprove bool, cfg *config.Config) *Registry {
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
	reg.Register(&SymbolTool{WorkspaceDir: workspaceDir})
	reg.Register(&BrowserTool{
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
	})

	return reg
}
