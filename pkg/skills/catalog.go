package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadCatalog scans all global and workspace directories for skills, agents, and rules
func LoadCatalog(claudeDir string) (*Catalog, error) {
	cat := NewCatalog()
	homeDir, _ := os.UserHomeDir()

	// 1. Skill Discovery Paths (Workspace overrides Global)
	skillDirs := []string{
		filepath.Join(homeDir, ".gemini", "config", "skills"),
		filepath.Join(homeDir, ".config", "claudekit", "skills"),
		filepath.Join(homeDir, ".claude", "skills"),
		filepath.Join(homeDir, ".mncode", "skills"),
		filepath.Join(homeDir, ".gemini", "antigravity-cli", "builtin", "skills"),
	}
	if claudeDir != "" {
		skillDirs = append(skillDirs, filepath.Join(claudeDir, "skills"))
		skillDirs = append(skillDirs, filepath.Join(filepath.Dir(claudeDir), ".mncode", "skills"))
	}

	for _, dir := range skillDirs {
		loaded, err := LoadSkills(dir)
		if err == nil && len(loaded) > 0 {
			for k, v := range loaded {
				cat.Skills[k] = v
			}
		}
	}

	// 2. Agent Discovery Paths
	agentDirs := []string{
		filepath.Join(homeDir, ".claude", "agents"),
	}
	if claudeDir != "" {
		agentDirs = append(agentDirs, filepath.Join(claudeDir, "agents"))
	}
	for _, dir := range agentDirs {
		loaded, err := LoadAgents(dir)
		if err == nil && len(loaded) > 0 {
			for k, v := range loaded {
				cat.Agents[k] = v
			}
		}
	}

	// 3. Rule Discovery Paths
	ruleDirs := []string{
		filepath.Join(homeDir, ".claude", "rules"),
	}
	if claudeDir != "" {
		ruleDirs = append(ruleDirs, filepath.Join(claudeDir, "rules"))
	}
	for _, dir := range ruleDirs {
		loaded, err := LoadRules(dir)
		if err == nil && len(loaded) > 0 {
			for k, v := range loaded {
				cat.Rules[k] = v
			}
		}
	}

	return cat, nil
}

// FormatSkillsCatalog formats the skills list for the system prompt
func (c *Catalog) FormatSkillsCatalog() string {
	if len(c.Skills) == 0 {
		return ""
	}

	// Deduplicate by FilePath
	seenFiles := make(map[string]bool)
	var uniqueSkills []*Skill
	for _, s := range c.Skills {
		if !seenFiles[s.FilePath] {
			seenFiles[s.FilePath] = true
			uniqueSkills = append(uniqueSkills, s)
		}
	}

	sort.Slice(uniqueSkills, func(i, j int) bool {
		return uniqueSkills[i].Name < uniqueSkills[j].Name
	})

	var sb strings.Builder
	sb.WriteString("<skills>\n")
	sb.WriteString("You can activate specialized 'skills' to help you with complex tasks.\n")
	sb.WriteString("To use a skill, you can read its SKILL.md file via 'view_file' or invoke the 'use_skill' tool.\n\n")
	sb.WriteString("Available skills:\n")

	for _, s := range uniqueSkills {
		sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", s.Name, s.FilePath, s.Description))
	}
	sb.WriteString("</skills>\n")

	return sb.String()
}

// FormatRules formats the rules for system prompt
func (c *Catalog) FormatRules() string {
	if len(c.Rules) == 0 {
		return ""
	}

	var names []string
	for name := range c.Rules {
		names = append(names, name)
	}
	sort.Strings(names)

	var sb strings.Builder
	sb.WriteString("<user_rules>\n")
	for _, name := range names {
		r := c.Rules[name]
		sb.WriteString(fmt.Sprintf("<RULE[%s]>\n%s\n</RULE[%s]>\n\n", r.Name, strings.TrimSpace(r.Content), r.Name))
	}
	sb.WriteString("</user_rules>\n")

	return sb.String()
}
