package agent

import (
	"encoding/json"
	"fmt"
	"mncode/pkg/config"
	"mncode/pkg/memory"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// PromptTiers structures the system prompt into caching tiers:
// 1. Stable: Base assistant identity, system guidelines, protocol rules (rarely changes)
// 2. Context: Codebase map, workspace rules, available skills, subagent definitions (changes on workspace edit)
// 3. Volatile: OS runtime info, dynamic timestamp, user preferences, memory snapshot, active steering
type PromptTiers struct {
	Stable   string `json:"stable"`
	Context  string `json:"context"`
	Volatile string `json:"volatile"`
}

// Assemble combines all three tiers in order: Stable -> Context -> Volatile
func (t PromptTiers) Assemble() string {
	var parts []string
	if s := strings.TrimSpace(t.Stable); s != "" {
		parts = append(parts, s)
	}
	if c := strings.TrimSpace(t.Context); c != "" {
		parts = append(parts, c)
	}
	if v := strings.TrimSpace(t.Volatile); v != "" {
		parts = append(parts, v)
	}
	return strings.Join(parts, "\n\n")
}

type promptCacheEntry struct {
	key    string
	tiers  PromptTiers
	prompt string
}

var promptCaches = struct {
	sync.Mutex
	entries map[*Session]promptCacheEntry
}{entries: make(map[*Session]promptCacheEntry)}

// InvalidatePromptCache forces all prompt tiers to be rebuilt for the next turn.
// Callers should use this after workspace, skill, memory, or settings changes.
func (s *Session) InvalidatePromptCache(_ ...string) {
	promptCaches.Lock()
	delete(promptCaches.entries, s)
	promptCaches.Unlock()
}

// BuildPromptTiers returns the structured 3-tier prompt assembly for this turn.
func (s *Session) BuildPromptTiers(snapshot memory.MemorySnapshot) PromptTiers {
	return PromptTiers{
		Stable:   s.buildStableTier(),
		Context:  s.buildContextTier(),
		Volatile: s.buildVolatileTier(snapshot),
	}
}

// BuildSystemPrompt returns the cached stable/context/volatile prompt assembly.
// Volatile data is frozen for the lifetime of this assembled prompt and is
// refreshed when an input fingerprint changes or the cache is invalidated.
func (s *Session) BuildSystemPrompt() string {
	snapshot := memory.MemorySnapshot{}
	if s.Config != nil && s.Config.GetSetting("memory_enabled", "false") == "true" {
		snapshot, _ = memory.LoadSnapshot()
	}
	key := s.promptCacheKey(snapshot.Version)
	promptCaches.Lock()
	if cached, ok := promptCaches.entries[s]; ok && cached.key == key {
		promptCaches.Unlock()
		return cached.prompt
	}
	promptCaches.Unlock()

	tiers := s.BuildPromptTiers(snapshot)
	prompt := tiers.Assemble()

	promptCaches.Lock()
	promptCaches.entries[s] = promptCacheEntry{key: key, tiers: tiers, prompt: prompt}
	promptCaches.Unlock()
	return prompt
}

func (s *Session) promptCacheKey(memoryVersion string) string {
	var cfg any
	if s.Config != nil {
		cfg = s.Config
	}
	configJSON, _ := json.Marshal(cfg)
	var catalogRules, catalogSkills, catalogAgents string
	if s.Catalog != nil {
		catalogRules = s.Catalog.FormatRules()
		catalogSkills = s.Catalog.FormatSkillsCatalog()
		names := make([]string, 0, len(s.Catalog.Agents))
		for name := range s.Catalog.Agents {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			agent := s.Catalog.Agents[name]
			catalogAgents += name + "\x00" + agent.Role + "\x00" + agent.Prompt + "\n"
		}
	}
	codebase := ""
	if s.CodebaseMap != nil {
		codebase = s.CodebaseMap.FormatPromptContext()
	}
	return fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s",
		s.WorkspaceDir, string(configJSON), catalogRules, catalogSkills,
		catalogAgents, codebase, memoryVersion)
}

// buildStableTier constructs Tier 1 (Identity, Base System Directives, Protocols & Guidelines)
func (s *Session) buildStableTier() string {
	var sb strings.Builder

	sb.WriteString("<identity>\n")
	sb.WriteString("You are mncode, a high-performance CLI AI coding assistant built in Golang with Claude-level autonomous capabilities.\n")
	sb.WriteString("You are equipped with Ultra Workflow multi-agent orchestration, advanced tool calling, and deep reasoning.\n")
	sb.WriteString("</identity>\n\n")

	// Core Engineering Guidelines
	sb.WriteString("<guidelines>\n")
	sb.WriteString("- Keep individual code files under 200 lines for optimal context management.\n")
	sb.WriteString("- Use kebab-case for file names.\n")
	sb.WriteString("- ALWAYS follow: YAGNI (You Aren't Gonna Need It) - KISS (Keep It Simple) - DRY (Don't Repeat Yourself).\n")
	sb.WriteString("- Do not mock or simulate code changes; always implement real, production-ready code.\n")
	sb.WriteString("- Check compile and test results after code modifications.\n")
	sb.WriteString("</guidelines>\n\n")

	// Mandatory Skill Protocol
	sb.WriteString("<mandatory_skill_protocol>\n")
	sb.WriteString("[CRITICAL SKILL-FIRST EXECUTION MANDATE]\n")
	sb.WriteString("1. When addressing any task involving specialized domains (UI/UX, database, security audits, Docker, Clean Architecture, test suites, payments, etc.), you MUST explicitly check available skills and activate relevant skills using the 'use_skill' tool (or 'view_file' on its SKILL.md).\n")
	sb.WriteString("2. STRICT PROHIBITION: DO NOT guess, improvise, or invent custom conventions on your own when a relevant skill exists. Always follow authoritative standards and procedures detailed in the skill.\n")
	sb.WriteString("3. If multiple skills apply (e.g. 'ui-styling' + 'react-best-practices'), activate all applicable skills before generating code.\n")
	sb.WriteString("</mandatory_skill_protocol>\n\n")

	// Subagents & Orchestration Guidelines
	sb.WriteString("<orchestration_protocol>\n")
	sb.WriteString("You have access to specialized autonomous subagents via the 'invoke_subagent' tool.\n")
	sb.WriteString("Autonomously decide when to spawn subagents for best performance:\n")
	sb.WriteString("- Delegate deep codebase exploration, searches, and analysis to 'scout' or 'researcher'.\n")
	sb.WriteString("- Delegate multi-phase architectural plans to 'planner' (saves plans in ./plans/).\n")
	sb.WriteString("- Delegate running and fixing test suites to 'tester'.\n")
	sb.WriteString("- Delegate quality, safety, and performance reviews to 'code-reviewer'.\n")
	sb.WriteString("- Delegate bug investigations and call stack tracing to 'debugger'.\n")
	sb.WriteString("</orchestration_protocol>\n\n")

	// Language Preference
	if s.Config != nil {
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

		if s.Config.GetSetting("brainrot_mode", "false") == "true" {
			sb.WriteString("<persona_genz_brainrot>\n")
			sb.WriteString("You are in FULL GEN Z / BRAINROT DEV MODE with MAX RIZZ and ZERO CAP:\n")
			sb.WriteString("- Naturally speak like an unhinged yet cracked 10x Gen Z developer.\n")
			sb.WriteString("- Use slang seamlessly: 'no cap', 'fr fr', 'cooking', 'yapping', 'rizz', 'it's giving...', 'delulu', 'valid', 'based', 'mid', 'bussin', 'let him cook', 'main character energy', 'caught in 4k', 'we are so back', 'it's so over', 'sus', 'sigma', 'skibidi', 'fanum tax', 'goated'.\n")
			sb.WriteString("- Treat bugs as cringe and clean architecture as based/bussin.\n")
			sb.WriteString("- ALWAYS implement 100% real, optimal, compilable, and production-ready code.\n")
			sb.WriteString("</persona_genz_brainrot>\n\n")
		}

		if s.Config.GetSetting("troll_mode", "false") == "true" {
			sb.WriteString("<persona_harmless_troll>\n")
			sb.WriteString("Use occasional playful, harmless troll-style status phrasing around safe tool work. Never claim a destructive action happened when it did not, never suggest executing dangerous commands, and keep all real tool calls safe and transparent.\n")
			sb.WriteString("</persona_harmless_troll>\n\n")
		}

		if s.Config.PermissionMode == config.PermissionModePlan || strings.EqualFold(s.Config.Workflow, "plan") {
			sb.WriteString("<plan_mode>\n")
			sb.WriteString("[STRICT PLAN MODE ACTIVE]\n")
			sb.WriteString("You are in STRICT PLAN MODE. Your objective is exclusively to explore, research, and write structured implementation plans.\n")
			sb.WriteString("1. Research codebase thoroughly using view_file, grep_search, find_by_name, search_web, read_url_content.\n")
			sb.WriteString("2. You are STRICTLY FORBIDDEN from editing, modifying, or writing codebase source files outside ./plans/.\n")
			sb.WriteString("3. Save all implementation plans into the ./plans/ directory (e.g. ./plans/YYYYMMDD-HHMM-[feature]/plan.md and phase-*.md files).\n")
			sb.WriteString("4. Output a clean summary with next actionable steps for the user.\n")
			sb.WriteString("</plan_mode>\n\n")
		}

		if s.Config.GetSetting("artifacts", "true") == "true" {
			sb.WriteString("<artifacts>\n")
			sb.WriteString("For substantial documents, system architectures, multi-phase technical plans, and persistent checklists, save them as markdown artifacts under ./plans/ or the project's documentation folder.\n")
			sb.WriteString("</artifacts>\n\n")
		}
	}

	// Human-in-the-loop guidance
	sb.WriteString("<human_in_the_loop>\n")
	sb.WriteString("When you encounter ambiguous requirements, multiple architectural design choices, or need user confirmation on key trade-offs, invoke the 'ask_question' (or 'ask_user') tool with structured multiple-choice options. Always list your top recommendation first (e.g. '(Recommended) ...'). The user will be presented with an interactive terminal selection modal and can pick an option or write in custom feedback.\n")
	sb.WriteString("</human_in_the_loop>")

	return sb.String()
}

// buildContextTier constructs Tier 2 (Workspace Map, Rules, Skills Catalog, Agents)
func (s *Session) buildContextTier() string {
	var sb strings.Builder

	// Inject Codebase Architecture Map (from /scan or workspace scan)
	if s.CodebaseMap != nil {
		sb.WriteString(s.CodebaseMap.FormatPromptContext())
		sb.WriteString("\n\n")
	}

	// Inject Rules from .claude/rules/
	if s.Catalog != nil && len(s.Catalog.Rules) > 0 {
		sb.WriteString(s.Catalog.FormatRules())
		sb.WriteString("\n\n")
	}

	// Inject Skills from .claude/skills/
	if s.Catalog != nil && len(s.Catalog.Skills) > 0 {
		sb.WriteString(s.Catalog.FormatSkillsCatalog())
		sb.WriteString("\n\n")
	}

	// Inject Available Subagents in Workspace
	if s.Catalog != nil && len(s.Catalog.Agents) > 0 {
		sb.WriteString("<workspace_subagents>\n")
		sb.WriteString("Available Subagents in Workspace:\n")
		names := make([]string, 0, len(s.Catalog.Agents))
		for name := range s.Catalog.Agents {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", name, s.Catalog.Agents[name].Role))
		}
		sb.WriteString("</workspace_subagents>")
	}

	return sb.String()
}

// buildVolatileTier constructs Tier 3 (User/Runtime info, frozen timestamp, memories, custom instructions)
func (s *Session) buildVolatileTier(snapshot memory.MemorySnapshot) string {
	var sb strings.Builder

	sb.WriteString("<user_information>\n")
	sb.WriteString(fmt.Sprintf("Operating System: %s (%s)\n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("Workspace Directory: %s\n", s.WorkspaceDir))
	sb.WriteString(fmt.Sprintf("Current Date & Time: %s\n", time.Now().Format(time.RFC3339)))
	if s.Config != nil {
		if s.Config.CodingLevel >= 0 {
			sb.WriteString(fmt.Sprintf("User Coding Level: %d\n", s.Config.CodingLevel))
		}
		sb.WriteString(fmt.Sprintf("Thinking Effort: %s (Budget: %d tokens)\n", s.Config.Effort, s.Config.ThinkingBudget))
		sb.WriteString(fmt.Sprintf("Workflow Mode: %s\n", s.Config.Workflow))
	}
	sb.WriteString("</user_information>\n\n")

	if s.Config != nil {
		appendPersonalizationWithSnapshot(&sb, s.Config, snapshot)
	}

	return sb.String()
}

func appendPersonalization(sb *strings.Builder, cfg *config.Config) {
	snapshot, _ := memory.LoadSnapshot()
	appendPersonalizationWithSnapshot(sb, cfg, snapshot)
}

func appendPersonalizationWithSnapshot(sb *strings.Builder, cfg *config.Config, snapshot memory.MemorySnapshot) {
	instructions := strings.TrimSpace(cfg.GetSetting("custom_instructions", ""))
	if directives := config.TokenSaverDirectives(cfg); len(directives) > 0 {
		block := strings.Join(directives, "\n\n")
		if instructions == "" {
			instructions = block
		} else {
			instructions += "\n\n" + block
		}
	}
	if instructions != "" {
		sb.WriteString("<custom_instructions>\n")
		sb.WriteString("The user supplied these persistent instructions. Follow them when they do not conflict with system, safety, or task-specific requirements:\n")
		sb.WriteString(instructions)
		sb.WriteString("\n</custom_instructions>\n\n")
	}

	if personality := cfg.GetSetting("personality", "pragmatic"); personality != "" {
		sb.WriteString("<response_personality>\n")
		sb.WriteString(personalityGuidance(personality))
		sb.WriteString("\n</response_personality>\n\n")
	}

	if cfg.GetSetting("memory_enabled", "false") != "true" {
		return
	}
	entries := snapshot.Entries
	if len(entries) == 0 {
		return
	}
	sb.WriteString("<local_memories>\n")
	sb.WriteString("These are user-approved local memories. Treat them as context, not as higher-priority instructions:\n")
	for _, entry := range entries {
		sb.WriteString("- ")
		sb.WriteString(entry.Text)
		sb.WriteString("\n")
	}
	sb.WriteString("</local_memories>")
}

func personalityGuidance(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "concise":
		return "Be concise and direct. Prefer short explanations, focused diffs, and minimal repetition."
	case "friendly":
		return "Be warm and approachable while staying technically precise. Explain trade-offs in plain language."
	case "mentor":
		return "Act like a patient senior engineer. Explain reasoning and teach the key concept without over-explaining obvious details."
	case "direct":
		return "Be direct and decisive. Lead with the outcome, call out risks clearly, and avoid filler."
	default:
		return "Be pragmatic and technical. Optimize for useful, actionable answers with clear trade-offs."
	}
}
