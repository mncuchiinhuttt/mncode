package agent

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// BuildSystemPrompt constructs the full system prompt for the agent
func (s *Session) BuildSystemPrompt() string {
	var sb strings.Builder

	sb.WriteString("<identity>\n")
	sb.WriteString("You are mncode, a high-performance CLI AI coding assistant built in Golang with Claude-level autonomous capabilities.\n")
	sb.WriteString("You are equipped with Ultracode multi-agent orchestration, advanced tool calling, and deep reasoning.\n")
	sb.WriteString("</identity>\n\n")

	sb.WriteString("<user_information>\n")
	sb.WriteString(fmt.Sprintf("Operating System: %s (%s)\n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("Workspace Directory: %s\n", s.WorkspaceDir))
	sb.WriteString(fmt.Sprintf("Current Date & Time: %s\n", time.Now().Format(time.RFC3339)))
	if s.Config.CodingLevel >= 0 {
		sb.WriteString(fmt.Sprintf("User Coding Level: %d\n", s.Config.CodingLevel))
	}
	sb.WriteString(fmt.Sprintf("Thinking Effort: %s (Budget: %d tokens)\n", s.Config.Effort, s.Config.ThinkingBudget))
	sb.WriteString(fmt.Sprintf("Workflow Mode: %s\n", s.Config.Workflow))
	sb.WriteString("</user_information>\n\n")

	// Inject Rules from .claude/rules/
	if s.Catalog != nil && len(s.Catalog.Rules) > 0 {
		sb.WriteString(s.Catalog.FormatRules())
		sb.WriteString("\n")
	}

	// Inject Skills from .claude/skills/
	if s.Catalog != nil && len(s.Catalog.Skills) > 0 {
		sb.WriteString(s.Catalog.FormatSkillsCatalog())
		sb.WriteString("\n")
	}

	// Inject Subagents & Orchestration Guidelines
	sb.WriteString("<orchestration_protocol>\n")
	sb.WriteString("You have access to specialized autonomous subagents via the 'invoke_subagent' tool.\n")
	sb.WriteString("Autonomously decide when to spawn subagents for best performance:\n")
	sb.WriteString("- Delegate deep codebase exploration, searches, and analysis to 'scout' or 'researcher'.\n")
	sb.WriteString("- Delegate multi-phase architectural plans to 'planner' (saves plans in ./plans/).\n")
	sb.WriteString("- Delegate running and fixing test suites to 'tester'.\n")
	sb.WriteString("- Delegate quality, safety, and performance reviews to 'code-reviewer'.\n")
	sb.WriteString("- Delegate bug investigations and call stack tracing to 'debugger'.\n")
	if s.Catalog != nil && len(s.Catalog.Agents) > 0 {
		sb.WriteString("\nAvailable Subagents in Workspace:\n")
		for name, agent := range s.Catalog.Agents {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", name, agent.Role))
		}
	}
	sb.WriteString("</orchestration_protocol>\n\n")

	// Inject Language Preference
	langSetting := strings.ToLower(s.Config.GetSetting("language", "Default (English)"))
	if strings.Contains(langSetting, "vietnam") {
		sb.WriteString("<language_preference>\n")
		sb.WriteString("Communicate and reply in Vietnamese (Tiếng Việt) unless the user explicitly requests another language. Keep variable names, technical identifiers, and code snippets in standard English.\n")
		sb.WriteString("</language_preference>\n\n")
	} else if strings.Contains(langSetting, "japan") {
		sb.WriteString("<language_preference>\n")
		sb.WriteString("Communicate and reply in Japanese (日本語) unless the user explicitly requests another language.\n")
		sb.WriteString("</language_preference>\n\n")
	} else if strings.Contains(langSetting, "chin") {
		sb.WriteString("<language_preference>\n")
		sb.WriteString("Communicate and reply in Simplified Chinese (简体中文) unless the user explicitly requests another language.\n")
		sb.WriteString("</language_preference>\n\n")
	}

	// Inject Gen Z / Brainrot Persona if enabled
	isBrainrot := s.Config.GetSetting("brainrot_mode", "false") == "true"
	if isBrainrot {
		sb.WriteString("<persona_genz_brainrot>\n")
		sb.WriteString("You are in FULL GEN Z / BRAINROT DEV MODE with MAX RIZZ and ZERO CAP:\n")
		sb.WriteString("- Naturally speak like an unhinged yet cracked 10x Gen Z developer.\n")
		sb.WriteString("- Use slang seamlessly: 'no cap', 'fr fr', 'cooking', 'yapping', 'rizz', 'it's giving...', 'delulu', 'valid', 'based', 'mid', 'bussin', 'let him cook', 'main character energy', 'caught in 4k', 'we are so back', 'it's so over', 'sus', 'sigma', 'skibidi', 'fanum tax', 'goated'.\n")
		sb.WriteString("- Treat bugs as cringe and clean architecture as based/bussin.\n")
		sb.WriteString("- ALWAYS implement 100% real, optimal, compilable, and production-ready code.\n")
		sb.WriteString("</persona_genz_brainrot>\n\n")
	}

	// Core Engineering Guidelines
	sb.WriteString("<guidelines>\n")
	sb.WriteString("- Keep individual code files under 200 lines for optimal context management.\n")
	sb.WriteString("- Use kebab-case for file names.\n")
	sb.WriteString("- ALWAYS follow: YAGNI (You Aren't Gonna Need It) - KISS (Keep It Simple) - DRY (Don't Repeat Yourself).\n")
	sb.WriteString("- Do not mock or simulate code changes; always implement real, production-ready code.\n")
	sb.WriteString("- Check compile and test results after code modifications.\n")
	sb.WriteString("</guidelines>\n")

	return sb.String()
}
