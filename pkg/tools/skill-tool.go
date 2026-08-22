package tools

import (
	"context"
	"fmt"
	"mncode/pkg/skills"
	"strings"
)

type SkillTool struct {
	Catalog *skills.Catalog
}

func (t *SkillTool) Name() string {
	return "use_skill"
}

func (t *SkillTool) Description() string {
	return "Activate a specialized ClaudeKit or workspace skill by name to read its instructions and workflows from SKILL.md"
}

func (t *SkillTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"skill_name": map[string]interface{}{
				"type":        "string",
				"description": "The name or alias of the skill to activate (e.g. 'frontend-design', 'plan', 'git', 'debug', 'ui-styling')",
			},
		},
		"required": []string{"skill_name"},
	}
}

func (t *SkillTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	skillName, _ := args["skill_name"].(string)
	if skillName == "" {
		return "", fmt.Errorf("skill_name argument is required")
	}

	if t.Catalog == nil {
		return "", fmt.Errorf("skills catalog not initialized")
	}

	name := strings.ToLower(strings.TrimSpace(skillName))
	skill, ok := t.Catalog.Skills[name]
	if !ok {
		for k, s := range t.Catalog.Skills {
			if strings.Contains(k, name) || strings.Contains(strings.ToLower(s.Name), name) {
				skill = s
				break
			}
		}
	}

	if skill == nil {
		return "", fmt.Errorf("skill '%s' not found. Available skills can be listed with /skills", skillName)
	}

	instructions := skill.GetInstructions()
	return fmt.Sprintf("=== Skill Activated: %s ===\nFile: %s\n\n%s", skill.Name, skill.FilePath, instructions), nil
}
