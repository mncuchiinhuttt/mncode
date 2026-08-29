package agent

import (
	"fmt"
	"mncode/pkg/provider"
	"mncode/pkg/skills"
	"strings"
)

// PreprocessSkillTags scans user prompt for /ck:... or /skill:... tags and activates them
func (s *Session) PreprocessSkillTags(userInput string) string {
	if s.Catalog == nil || len(s.Catalog.Skills) == 0 {
		return userInput
	}

	parts := strings.Fields(userInput)
	var remainingWords []string
	activatedAny := false

	for _, word := range parts {
		if strings.HasPrefix(word, "/") {
			rawCmd := strings.ToLower(word)
			if rawCmd == "/help" || rawCmd == "/exit" || rawCmd == "/quit" || rawCmd == "/model" ||
				rawCmd == "/effort" || rawCmd == "/workflow" || rawCmd == "/agents" || rawCmd == "/context" ||
				rawCmd == "/compact" || rawCmd == "/clear" || rawCmd == "/status" || rawCmd == "/config" ||
				rawCmd == "/sync" || rawCmd == "/feedback" || rawCmd == "/btw" || rawCmd == "/brainrot" ||
				rawCmd == "/export" || rawCmd == "/trajectory" || rawCmd == "/sharegpt" || rawCmd == "/export-training" {
				remainingWords = append(remainingWords, word)
				continue
			}

			skillName := strings.TrimPrefix(word, "/ck:")
			skillName = strings.TrimPrefix(skillName, "/skill:")
			skillName = strings.TrimPrefix(skillName, "/")
			skillName = strings.ToLower(strings.TrimSpace(skillName))

			var matched *skills.Skill
			if sk, ok := s.Catalog.Skills[skillName]; ok {
				matched = sk
			} else if sk, ok := s.Catalog.Skills["ck:"+skillName]; ok {
				matched = sk
			}

			if matched != nil {
				activatedAny = true
				instr := matched.GetInstructions()
				fmt.Printf("\n\033[1;32m[Skill Activated]\033[0m %s (%s)\n",
					matched.Name, matched.Description)

				if instr != "" {
					appendHistory(s, provider.Message{
						Role:    provider.RoleSystem,
						Content: fmt.Sprintf("<activated_skill name=\"%s\">\n%s\n</activated_skill>", matched.Name, instr),
					})
				}
				continue
			}
		}
		remainingWords = append(remainingWords, word)
	}

	cleanedPrompt := strings.Join(remainingWords, " ")
	if activatedAny && strings.TrimSpace(cleanedPrompt) == "" {
		cleanedPrompt = "Skill activated and loaded into context. Please follow the skill instructions for subsequent tasks."
	}
	return cleanedPrompt
}
