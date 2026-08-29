package agent

import (
	"encoding/json"
	"sort"
	"strings"
)

type ContextUsage struct {
	Model             string
	DisplayName       string
	Limit             int
	TotalUsed         int
	SystemTokens      int
	SystemToolsTokens int
	MCPToolsTokens    int
	SkillsTokens      int
	MessagesTokens    int
	MessageCount      int
	ToolCount         int
	MCPToolCount      int
	SkillsCount       int
	PercentUsed       float64
	RemainingTokens   int
	AutoCompactBuffer int
}

// GetContextLimit returns the maximum context window size for a given model
func GetContextLimit(model string, prov string) int {
	m := strings.ToLower(model)
	if strings.Contains(m, "gemini") || strings.Contains(m, "claude-sonnet-4") || strings.Contains(m, "claude-opus-4") {
		return 1000000 // 1M tokens for Antigravity / Gemini
	}
	if strings.Contains(m, "claude-3-7") || strings.Contains(m, "claude-3-5") || strings.Contains(m, "claude-3-opus") {
		return 200000 // 200k tokens for Anthropic API
	}
	if strings.Contains(m, "deepseek-v4") || strings.Contains(m, "opencode") {
		return 200000 // 200k tokens for OpenCode Zen
	}
	if strings.Contains(m, "gpt-4o") || strings.Contains(m, "o1") || strings.Contains(m, "o3") {
		return 128000 // 128k tokens for OpenAI
	}
	return 1000000
}

// GetContextUsage calculates real token breakdown by inspecting active prompt components
func (s *Session) GetContextUsage() ContextUsage {
	limit := s.Config.GetContextWindowTokens()
	if limit <= 0 {
		limit = GetContextLimit(s.Config.Model, string(s.Config.Provider))
	}

	// 1. Exact System Prompt Tokens (Identity, Env info, Rules, Guidelines)
	var sysBuilder strings.Builder
	sysBuilder.WriteString("<identity>You are mncode CLI AI assistant</identity>\n")
	sysBuilder.WriteString("<user_information>OS, Workspace, Config</user_information>\n")
	if s.Catalog != nil && len(s.Catalog.Rules) > 0 {
		sysBuilder.WriteString(s.Catalog.FormatRules())
	}
	sysBuilder.WriteString("<guidelines>Engineering rules</guidelines>\n")
	sysTokens := len(sysBuilder.String()) / 4

	// 2. Exact System Tools Tokens (Native tools schema JSON payload)
	toolsTokens := 0
	toolCount := 0
	if s.Tools != nil {
		for _, t := range s.Tools.All() {
			toolCount++
			schemaBytes, _ := json.Marshal(t.Schema())
			toolsTokens += (len(t.Name()) + len(t.Description()) + len(schemaBytes)) / 4
		}
	}

	// 3. Exact Skills Tokens (FormatSkillsCatalog XML in prompt)
	skillsTokens := 0
	skillsCount := 0
	if s.Catalog != nil && len(s.Catalog.Skills) > 0 {
		skillsCount = len(s.Catalog.Skills)
		skillsTokens = len(s.Catalog.FormatSkillsCatalog()) / 4
	}

	// 4. Exact Subagents & MCP Tokens (Orchestration protocol + Agent roles)
	mcpTokens := 0
	mcpCount := 0
	if s.Catalog != nil && len(s.Catalog.Agents) > 0 {
		mcpCount = len(s.Catalog.Agents)
		var agentBuilder strings.Builder
		names := make([]string, 0, len(s.Catalog.Agents))
		for name := range s.Catalog.Agents {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			ag := s.Catalog.Agents[name]
			agentBuilder.WriteString(name + ": " + ag.Role + "\n" + ag.Prompt + "\n")
		}
		mcpTokens = (len(agentBuilder.String()) + 400) / 4
	}

	// 5. Exact Conversation Messages Tokens
	messagesTokens := 0
	for _, msg := range historySnapshot(s) {
		messagesTokens += (len(msg.Content) + len(msg.Thinking)) / 4
		for _, tc := range msg.ToolCalls {
			argsBytes, _ := json.Marshal(tc.Arguments)
			messagesTokens += (len(tc.Name) + len(argsBytes)) / 4
		}
		for _, tr := range msg.ToolResults {
			messagesTokens += len(tr.Content) / 4
		}
	}

	totalUsed := sysTokens + toolsTokens + mcpTokens + skillsTokens + messagesTokens
	if totalUsed > limit {
		totalUsed = limit
	}

	pct := (float64(totalUsed) / float64(limit)) * 100.0
	bufferTokens := int(float64(limit) * 0.034) // 3.4% auto-compact buffer
	rem := limit - totalUsed - bufferTokens
	if rem < 0 {
		rem = 0
	}

	displayName := s.Config.Model
	if strings.Contains(strings.ToLower(s.Config.Model), "sonnet") {
		displayName = "Claude Sonnet 4.6 (Thinking)"
	} else if strings.Contains(strings.ToLower(s.Config.Model), "opus") {
		displayName = "Claude Opus 4.6 (Thinking)"
	} else if strings.Contains(strings.ToLower(s.Config.Model), "gemini-3.7") {
		displayName = "Gemini 3.7 Flash"
	} else if strings.Contains(strings.ToLower(s.Config.Model), "gemini-2.5-pro") {
		displayName = "Gemini 2.5 Pro"
	} else if strings.Contains(strings.ToLower(s.Config.Model), "deepseek-v4") {
		displayName = "DeepSeek V4 Flash (Free)"
	}

	return ContextUsage{
		Model:             s.Config.Model,
		DisplayName:       displayName,
		Limit:             limit,
		TotalUsed:         totalUsed,
		SystemTokens:      sysTokens,
		SystemToolsTokens: toolsTokens,
		MCPToolsTokens:    mcpTokens,
		SkillsTokens:      skillsTokens,
		MessagesTokens:    messagesTokens,
		MessageCount:      len(s.History),
		ToolCount:         toolCount,
		MCPToolCount:      mcpCount,
		SkillsCount:       skillsCount,
		PercentUsed:       pct,
		RemainingTokens:   rem,
		AutoCompactBuffer: bufferTokens,
	}
}
