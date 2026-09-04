package combos

import "time"

// DefaultPresets returns the 4 standard built-in combos.
func DefaultPresets() []Combo {
	now := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)
	return []Combo{
		{
			ID:          "feature-delivery",
			Name:        "Full-Stack Feature Delivery",
			Description: "Plan, design UI, implement with worktree isolation, test, and conduct pre-merge review",
			Mode:        ModePipeline,
			IsBuiltin:   true,
			CreatedAt:   now,
			UpdatedAt:   now,
			Members: []ComboMember{
				{ID: "m1", Role: RolePlanner, BaseAgent: "planner", Model: "auto", FallbackModel: "auto", PromptOverlay: "Decompose user requirements into actionable verification phases."},
				{ID: "m2", Role: RoleDesigner, BaseAgent: "coder", Model: "auto", FallbackModel: "auto", PromptOverlay: "Refine UI/UX interfaces, Tailwind CSS styling, and responsive layout.", Permissions: []string{"edit_files"}},
				{ID: "m3", Role: RoleCoder, BaseAgent: "coder", Model: "auto", FallbackModel: "auto", IsolatedWorktree: true, PromptOverlay: "Implement core business logic cleanly adhering to SOLID principles.", Permissions: []string{"edit_files", "bash", "lsp"}},
				{ID: "m4", Role: RoleTester, BaseAgent: "tester", Model: "auto", FallbackModel: "auto", PromptOverlay: "Execute unit and regression tests, verifying that all tests pass 100%.", Permissions: []string{"bash"}},
				{ID: "m5", Role: RoleCodeReviewer, BaseAgent: "code-reviewer", Model: "auto", FallbackModel: "auto", PromptOverlay: "Conduct final security and quality audit on the diff.", Permissions: []string{"read_only"}},
			},
		},
		{
			ID:              "critic-refactor",
			Name:            "Architectural Debate & Refactoring",
			Description:     "Scout hotspots, debate proposed architecture with Critic, refactor, and verify regressions",
			Mode:            ModeDebate,
			MaxDebateRounds: 2,
			IsBuiltin:       true,
			CreatedAt:       now,
			UpdatedAt:       now,
			Members: []ComboMember{
				{ID: "d1", Role: RoleScout, BaseAgent: "scout", Model: "auto", FallbackModel: "auto", PromptOverlay: "Identify complexity hotspots, code smells, and coupling."},
				{ID: "d2", Role: RoleArchitect, BaseAgent: "planner", Model: "auto", FallbackModel: "auto", PromptOverlay: "Propose a decoupled, clean architecture refactoring plan."},
				{ID: "d3", Role: RoleAdvisor, BaseAgent: "code-reviewer", Model: "auto", FallbackModel: "auto", PromptOverlay: "Act as devil's advocate. Challenge assumptions, edge cases, and performance."},
				{ID: "d4", Role: RoleRefactorer, BaseAgent: "coder", Model: "auto", FallbackModel: "auto", IsolatedWorktree: true, PromptOverlay: "Implement the consensus refactoring keeping all files under 200 lines.", Permissions: []string{"edit_files", "lsp"}},
				{ID: "d5", Role: RoleTester, BaseAgent: "tester", Model: "auto", FallbackModel: "auto", PromptOverlay: "Run full regression test suite to ensure zero breaking changes.", Permissions: []string{"bash"}},
			},
		},
		{
			ID:          "speed-fix",
			Name:        "Lightning Bug Fixer",
			Description: "Rapid root-cause localization, sonic surgical edit, and quick regression verification",
			Mode:        ModePipeline,
			IsBuiltin:   true,
			CreatedAt:   now,
			UpdatedAt:   now,
			Members: []ComboMember{
				{ID: "s1", Role: RoleScout, BaseAgent: "scout", Model: "auto", FallbackModel: "auto", PromptOverlay: "Quickly pinpoint the exact error location and relevant line range."},
				{ID: "s2", Role: RoleWorker, BaseAgent: "sonic", Model: "auto", FallbackModel: "auto", PromptOverlay: "Apply the minimal, surgical fix with zero unnecessary edits.", Permissions: []string{"edit_files", "bash"}},
				{ID: "s3", Role: RoleTester, BaseAgent: "tester", Model: "auto", FallbackModel: "auto", PromptOverlay: "Run focused reproduction test to confirm defect is resolved.", Permissions: []string{"bash"}},
			},
		},
		{
			ID:          "security-audit",
			Name:        "Parallel Security & AppSec Sweep",
			Description: "Concurrent vulnerability scanning and secrets audit with synthesized mitigation patches",
			Mode:        ModeFanOut,
			IsBuiltin:   true,
			CreatedAt:   now,
			UpdatedAt:   now,
			Members: []ComboMember{
				{ID: "a1", Role: RoleSecurityAuditor, BaseAgent: "code-reviewer", Model: "auto", FallbackModel: "auto", IsolatedWorktree: true, PromptOverlay: "Audit OWASP Top 10 vulnerabilities, injection flaws, and auth bypasses."},
				{ID: "a2", Role: RoleScout, BaseAgent: "scout", Model: "auto", FallbackModel: "auto", IsolatedWorktree: true, PromptOverlay: "Scan for hardcoded API keys, unencrypted secrets, and sensitive files."},
				{ID: "a3", Role: RoleAdvisor, BaseAgent: "code-reviewer", Model: "auto", FallbackModel: "auto", PromptOverlay: "Assess risk severity and propose remediation priority matrix."},
				{ID: "a4", Role: RoleRefactorer, BaseAgent: "coder", Model: "auto", FallbackModel: "auto", PromptOverlay: "Apply sanitized mitigation patches.", Permissions: []string{"edit_files"}},
			},
		},
	}
}
