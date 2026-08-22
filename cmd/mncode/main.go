package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"mncode/pkg/accounts"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"mncode/pkg/mcp"
	"mncode/pkg/provider"
	"mncode/pkg/skills"
	"mncode/pkg/stats"
	"mncode/pkg/tools"
	"mncode/pkg/ui"
)

func main() {
	var (
		workspaceFlag    = flag.String("w", ".", "Workspace directory path")
		modelFlag        = flag.String("m", "", "Model name override (e.g. claude-3-7-sonnet-20250219)")
		providerFlag     = flag.String("p", "", "Provider override (anthropic, openrouter, openai, gemini)")
		autoApprove      = flag.Bool("y", false, "Auto-approve all tool executions")
		bypassPerms      = flag.Bool("dangerously-skip-permissions", false, "Bypass all permission prompts for tool execution")
		bypassPermsShort = flag.Bool("dangerously-skip-permission", false, "Bypass all permission prompts for tool execution")
		promptFlag       = flag.String("e", "", "Execute a single prompt non-interactively")
		resumeFlag       = flag.Bool("resume", false, "Resume previous conversation session")
		resumeShort      = flag.Bool("r", false, "Resume previous conversation session")
		resumeID         = flag.String("resume-id", "", "Resume specific session ID")
		versionFlag      = flag.Bool("v", false, "Show mncode version")
		versionLong      = flag.Bool("version", false, "Show mncode version")
		upgradeFlag      = flag.Bool("upgrade", false, "Update mncode to latest release")
		updateFlag       = flag.Bool("update", false, "Update mncode to latest release")
		scanFlag         = flag.Bool("scan", false, "Deep scan and print codebase architecture map")
	)
	flag.Parse()

	if *versionFlag || *versionLong || (len(os.Args) > 1 && os.Args[1] == "version") {
		ui.ShowVersionInfo()
		return
	}

	if *upgradeFlag || *updateFlag || (len(os.Args) > 1 && (os.Args[1] == "upgrade" || os.Args[1] == "update")) {
		ui.HandleUpdateCommand(nil, nil)
		return
	}

	if *scanFlag || (len(os.Args) > 1 && os.Args[1] == "scan") {
		ui.HandleScanCommand(nil, &agent.Session{WorkspaceDir: *workspaceFlag})
		return
	}

	// 1. Load configuration
	cfg, err := config.LoadConfig(*workspaceFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if *modelFlag != "" {
		cfg.Model = *modelFlag
	}
	if *providerFlag != "" {
		cfg.Provider = config.ProviderType(*providerFlag)
	}
	ui.SetTheme(cfg.GetSetting("theme", "pastel-pink"))
	if *bypassPerms || *bypassPermsShort {
		cfg.PermissionMode = config.PermissionModeBypass
		cfg.AutoApprove = true
	} else if *autoApprove {
		cfg.PermissionMode = config.PermissionModeAuto
		cfg.AutoApprove = true
	}

	// 2. Initialize Account Store & 9router-style Router
	accStore, _ := accounts.NewStore("")
	accRouter := accounts.NewRouter(accStore)

	// Auto-import local credentials if pool is empty
	if len(accStore.Accounts) == 0 {
		_, _ = accounts.ImportAntigravityDefaultCreds(accStore)
		_, _ = accounts.ImportCodexCredentials(accStore)
	}

	// 3. Discover skills, agents, rules from .claude
	catalog, _ := skills.LoadCatalog(cfg.ClaudeDir)

	// 4. Auto-configure Provider from multi-account pool if not set in .env
	if cfg.APIKey == "" {
		if cfg.Provider == config.ProviderOpenCode {
			cfg.APIKey = "public"
			if cfg.BaseURL == "" {
				cfg.BaseURL = "https://opencode.ai/zen/v1"
			}
		} else if acc, err := accRouter.GetNextAccount(accounts.ProviderTypeAntigravity); err == nil && acc != nil {
			cfg.APIKey = acc.AccessToken
			if cfg.Provider == "" {
				cfg.Provider = config.ProviderAntigravity
			}
			if cfg.Model == "" {
				cfg.Model = "gemini-3.7-flash-high"
			}
		} else if acc, err := accRouter.GetNextAccount(accounts.ProviderTypeCodex); err == nil && acc != nil {
			cfg.APIKey = acc.AccessToken
			if cfg.Provider == "" {
				cfg.Provider = config.ProviderOpenAI
			}
			if cfg.Model == "" {
				cfg.Model = "gpt-4o"
			}
		}
	}

	var llmProvider provider.Provider
	if cfg.APIKey != "" {
		llmProvider, _ = provider.NewProvider(cfg)
	}

	// 5. Initialize Tools, MCP & UI
	toolRegistry := tools.DefaultRegistry(cfg.WorkspaceDir, cfg.AutoApprove)
	mcpMgr := mcp.NewManager(cfg.WorkspaceDir)
	termUI := ui.NewTerminalUI(cfg.AutoApprove)
	tokenTracker := stats.NewTracker()

	session := &agent.Session{
		ID:           "mncode-main",
		WorkspaceDir: cfg.WorkspaceDir,
		Config:       cfg,
		Provider:     llmProvider,
		Tools:        toolRegistry,
		Catalog:      catalog,
		Accounts:     accStore,
		Router:       accRouter,
		Tracker:      tokenTracker,
		Subagents:    agent.NewSubagentRegistry(),
		MCP:          mcpMgr,
		UI:           termUI,
	}

	// Start MCP servers in background and register tools
	go func() {
		ctx := context.Background()
		mcpMgr.StartAll(ctx)
		tools.RegisterMCPTools(toolRegistry, mcpMgr, ctx)
	}()

	// Register invoke_subagent and use_skill tools
	subRunner := &agent.SubagentRunner{ParentSession: session}
	toolRegistry.Register(&tools.SubagentTool{
		Invoker: subRunner.Run,
	})
	toolRegistry.Register(&tools.SkillTool{
		Catalog: catalog,
	})

	// Restore previous session if --resume requested
	if *resumeFlag || *resumeShort || *resumeID != "" {
		if *resumeID != "" {
			if saved, err := agent.LoadSavedSession(*resumeID); err == nil {
				session.Restore(saved)
				fmt.Printf("\n%s Resumed session: %s (%d turns)\n", ui.BoldGreen("[Resumed]"), ui.Bold(saved.Title), saved.Turns)
				ui.RenderResumedHistory(saved.Messages)
			} else {
				fmt.Printf("\n%s Failed to load session '%s': %v\n\n", ui.BoldRed("[Error]"), *resumeID, err)
			}
		} else {
			if latest, err := agent.GetLatestSavedSession(); err == nil {
				session.Restore(latest)
				fmt.Printf("\n%s Resumed latest session: %s (%d turns)\n", ui.BoldGreen("[Resumed]"), ui.Bold(latest.Title), latest.Turns)
				ui.RenderResumedHistory(latest.Messages)
			} else {
				fmt.Printf("\n%s No previous sessions found to resume.\n\n", ui.BoldYellow("[Notice]"))
			}
		}
	}

	// 6. Run Non-interactive or REPL
	if *promptFlag != "" {
		if strings.HasPrefix(*promptFlag, "/") {
			if ui.HandleSlashCommand(*promptFlag, session) {
				return
			}
		}
		ctx := context.Background()
		if err := session.ProcessUserInput(ctx, *promptFlag); err != nil {
			fmt.Fprintf(os.Stderr, "\nExecution error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Push today's usage telemetry once per day, if a sync key is configured.
	// Only for the interactive REPL — one-shot `-e` invocations (which may
	// themselves be a scripted `/sync`) shouldn't race a background push.
	go ui.MaybeAutoSyncDaily(session)

	ui.RunREPL(session)
}
