package skills

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadSkills scans a skills directory and parses all SKILL.md files
func LoadSkills(skillsDir string) (map[string]*Skill, error) {
	skills := make(map[string]*Skill)

	info, err := os.Stat(skillsDir)
	if err != nil || !info.IsDir() {
		return skills, nil
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return skills, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(skillsDir, entry.Name())
		skillFile := filepath.Join(skillDir, "SKILL.md")

		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		var skill Skill
		body, err := ParseFrontmatter(data, &skill)
		if err != nil || skill.Name == "" {
			skill.Name = entry.Name()
			if skill.Description == "" {
				skill.Description = "Skill from " + entry.Name()
			}
		}

		skill.Name = strings.TrimPrefix(strings.TrimSpace(skill.Name), "ck:")
		skill.Directory = skillDir
		skill.FilePath = skillFile
		skill.Body = body

		// Register clean names and lowercase aliases without ck: prefix
		nameClean := strings.ToLower(skill.Name)
		entryClean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(entry.Name()), "ck:"))

		skills[skill.Name] = &skill
		skills[nameClean] = &skill
		skills[entryClean] = &skill
		skills["ck:"+entryClean] = &skill
		skills["ck:"+nameClean] = &skill
	}

	return skills, nil
}

// GetSkillInstructions returns the loaded markdown instructions for a skill
func (s *Skill) GetInstructions() string {
	if s.Body != "" {
		return s.Body
	}
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		return ""
	}
	body, _ := ParseFrontmatter(data, nil)
	return strings.TrimSpace(body)
}
