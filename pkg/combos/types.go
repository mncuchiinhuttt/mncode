package combos

import "time"

// ExecutionMode defines how members in a combo coordinate.
type ExecutionMode string

const (
	ModePipeline ExecutionMode = "pipeline"
	ModeDebate   ExecutionMode = "debate"
	ModeFanOut   ExecutionMode = "fan_out"
)

// Standard role constants for the 16 core roles.
const (
	RolePlanner         = "planner"
	RoleArchitect       = "architect"
	RoleAdvisor         = "advisor"
	RoleDesigner        = "designer"
	RoleScout           = "scout"
	RoleResearcher      = "researcher"
	RoleCoder           = "coder"
	RoleWorker          = "worker"
	RoleRefactorer      = "refactorer"
	RoleTester          = "tester"
	RoleDebugger        = "debugger"
	RoleSecurityAuditor = "security-auditor"
	RoleCodeReviewer    = "code-reviewer"
	RoleDBSpecialist    = "db-specialist"
	RoleDevOps          = "devops"
	RoleDocsManager     = "docs-manager"
)

// RoleMeta describes metadata and defaults for a standard role.
type RoleMeta struct {
	Role                 string   `json:"role"`
	Title                string   `json:"title"`
	Description          string   `json:"description"`
	DefaultBaseAgent     string   `json:"defaultBaseAgent"`
	AutoPrimaryModel     string   `json:"autoPrimaryModel"`
	AutoFallbackModel    string   `json:"autoFallbackModel"`
	DefaultPermissions   []string `json:"defaultPermissions"`
	RequiresWorktreeBase bool     `json:"requiresWorktreeBase"`
}

// ComboMember represents a single participating agent within a Combo.
type ComboMember struct {
	ID               string   `json:"id"`
	Role             string   `json:"role"`
	BaseAgent        string   `json:"baseAgent"`
	PromptOverlay    string   `json:"promptOverlay,omitempty"`
	Model            string   `json:"model,omitempty"`         // "auto", specific model, or empty
	FallbackModel    string   `json:"fallbackModel,omitempty"` // "auto", specific model, or empty
	ThinkingBudget   int      `json:"thinkingBudget,omitempty"`
	Permissions      []string `json:"permissions,omitempty"`
	IsolatedWorktree bool     `json:"isolatedWorktree,omitempty"`
}

// Combo represents a multi-agent swarm/pipeline preset or custom workflow.
type Combo struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	Mode            ExecutionMode `json:"mode"`
	MaxDebateRounds int           `json:"maxDebateRounds,omitempty"` // For debate mode (default: 2)
	Members         []ComboMember `json:"members"`
	IsBuiltin       bool          `json:"isBuiltin,omitempty"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

// StandardRoles returns the catalog of 16 official roles with metadata.
func StandardRoles() []RoleMeta {
	return []RoleMeta{
		{Role: RolePlanner, Title: "Principal Planner", Description: "Decomposes tasks into verified phases", DefaultBaseAgent: "planner", AutoPrimaryModel: "claude-sonnet-4-6", AutoFallbackModel: "gemini-3.7-flash-high", DefaultPermissions: []string{"read_only"}},
		{Role: RoleArchitect, Title: "System Architect", Description: "Designs system structure & API contracts", DefaultBaseAgent: "planner", AutoPrimaryModel: "claude-opus-4-6-thinking", AutoFallbackModel: "gemini-pro-agent", DefaultPermissions: []string{"read_only"}},
		{Role: RoleAdvisor, Title: "Technical Critic", Description: "Challenges assumptions & identifies edge cases", DefaultBaseAgent: "code-reviewer", AutoPrimaryModel: "deepseek-reasoner", AutoFallbackModel: "gemini-3.7-flash-high", DefaultPermissions: []string{"read_only"}},
		{Role: RoleDesigner, Title: "UI/UX Designer", Description: "Builds UI components & responsive styles", DefaultBaseAgent: "coder", AutoPrimaryModel: "claude-sonnet-4-6", AutoFallbackModel: "gemini-3.7-flash-high", DefaultPermissions: []string{"edit_files"}},
		{Role: RoleScout, Title: "Codebase Scout", Description: "Maps codebase and call graphs rapidly", DefaultBaseAgent: "scout", AutoPrimaryModel: "gemini-3.7-flash-high", AutoFallbackModel: "gemini-2.5-flash", DefaultPermissions: []string{"read_only", "lsp"}},
		{Role: RoleResearcher, Title: "Tech Researcher", Description: "Traverses docs & search with citations", DefaultBaseAgent: "researcher", AutoPrimaryModel: "gemini-3.7-flash-high", AutoFallbackModel: "o3-mini", DefaultPermissions: []string{"web_search", "read_only"}},
		{Role: RoleCoder, Title: "Principal Coder", Description: "Core logic and robust implementation", DefaultBaseAgent: "coder", AutoPrimaryModel: "claude-sonnet-4-6", AutoFallbackModel: "gemini-3.7-flash-high", DefaultPermissions: []string{"edit_files", "bash", "lsp"}},
		{Role: RoleWorker, Title: "Sonic Helper", Description: "Utility tasks, mocks & boilerplates", DefaultBaseAgent: "sonic", AutoPrimaryModel: "gemini-2.5-flash", AutoFallbackModel: "gemini-3.7-flash-high", DefaultPermissions: []string{"edit_files", "bash"}},
		{Role: RoleRefactorer, Title: "Clean Code Specialist", Description: "Reduces complexity & separates concerns", DefaultBaseAgent: "coder", AutoPrimaryModel: "claude-sonnet-4-6", AutoFallbackModel: "gemini-3.7-flash-high", DefaultPermissions: []string{"edit_files", "lsp"}},
		{Role: RoleTester, Title: "Test Engineer", Description: "Runs and authors comprehensive test suites", DefaultBaseAgent: "tester", AutoPrimaryModel: "gemini-3.7-flash-high", AutoFallbackModel: "o3-mini", DefaultPermissions: []string{"bash", "edit_files"}},
		{Role: RoleDebugger, Title: "Root-Cause Debugger", Description: "Investigates trace, logs and DAP breakpoints", DefaultBaseAgent: "debugger", AutoPrimaryModel: "claude-sonnet-4-6", AutoFallbackModel: "gemini-pro-agent", DefaultPermissions: []string{"bash", "dap", "read_only"}},
		{Role: RoleSecurityAuditor, Title: "AppSec Auditor", Description: "Audits OWASP Top 10, secrets & permissions", DefaultBaseAgent: "code-reviewer", AutoPrimaryModel: "claude-opus-4-6-thinking", AutoFallbackModel: "deepseek-reasoner", DefaultPermissions: []string{"read_only"}},
		{Role: RoleCodeReviewer, Title: "Lead Gatekeeper", Description: "Pre-merge diff correctness check", DefaultBaseAgent: "code-reviewer", AutoPrimaryModel: "claude-sonnet-4-6", AutoFallbackModel: "gemini-3.7-flash-high", DefaultPermissions: []string{"read_only"}},
		{Role: RoleDBSpecialist, Title: "Database Engineer", Description: "Database schemas, queries & migrations", DefaultBaseAgent: "coder", AutoPrimaryModel: "claude-sonnet-4-6", AutoFallbackModel: "gemini-3.7-flash-high", DefaultPermissions: []string{"edit_files", "bash"}},
		{Role: RoleDevOps, Title: "Cloud & CI/CD Engineer", Description: "Docker, CI/CD and deployment automation", DefaultBaseAgent: "coder", AutoPrimaryModel: "gemini-3.7-flash-high", AutoFallbackModel: "o3-mini", DefaultPermissions: []string{"edit_files", "bash"}},
		{Role: RoleDocsManager, Title: "Docs Maintainer", Description: "Maintains architecture docs & changelogs", DefaultBaseAgent: "docs-manager", AutoPrimaryModel: "gemini-2.5-flash", AutoFallbackModel: "gemini-3.7-flash-high", DefaultPermissions: []string{"edit_files"}},
	}
}

// FindRoleMeta returns metadata for a role if standard, or a custom placeholder.
func FindRoleMeta(role string) RoleMeta {
	for _, meta := range StandardRoles() {
		if meta.Role == role {
			return meta
		}
	}
	return RoleMeta{
		Role:              role,
		Title:             role,
		Description:       "Custom user-defined subagent role",
		DefaultBaseAgent:  "coder",
		AutoPrimaryModel:  "claude-sonnet-4-6",
		AutoFallbackModel: "gemini-3.7-flash-high",
	}
}
