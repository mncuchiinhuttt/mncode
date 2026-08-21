package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/provider"
	"mncode/pkg/skills"
	"strings"
)

// HandleSkillCommand handles /skill or /skills command
func HandleSkillCommand(parts []string, s *agent.Session) {
	if len(parts) > 1 && parts[0] != "/skills" {
		skillQuery := strings.TrimSpace(strings.Join(parts[1:], " "))
		ActivateSkillByName(s, skillQuery)
		return
	}
	OpenInteractiveSkillsBrowser(s)
}

// ActivateSkillByName activates a skill and injects its instructions into the session
func ActivateSkillByName(s *agent.Session, name string) {
	if s.Catalog == nil || len(s.Catalog.Skills) == 0 {
		fmt.Printf("%s No skills loaded.\n", BoldRed("[Error]"))
		return
	}

	q := strings.ToLower(name)
	var matched *skills.Skill
	if sk, ok := s.Catalog.Skills[q]; ok {
		matched = sk
	} else {
		for k, sk := range s.Catalog.Skills {
			if strings.Contains(k, q) || strings.Contains(strings.ToLower(sk.Name), q) {
				matched = sk
				break
			}
		}
	}

	if matched == nil {
		fmt.Printf("%s Skill '%s' not found. Type '/skills' to view all available skills.\n", BoldRed("[Error]"), name)
		return
	}

	instr := matched.GetInstructions()
	fmt.Printf("\n%s Activated Skill: %s\n  %s %s\n  %s %s\n\n",
		BoldGreen("[Skill Activated]"),
		Bold(matched.Name),
		GrayText("Path:"), matched.FilePath,
		GrayText("Summary:"), matched.Description)

	// Inject into session history as system context instruction
	if instr != "" {
		s.History = append(s.History, provider.Message{
			Role:    "system",
			Content: fmt.Sprintf("<activated_skill name=\"%s\">\n%s\n</activated_skill>", matched.Name, instr),
		})
	}
}

func showSkillsList(s *agent.Session) {
	if s.Catalog == nil {
		fmt.Println("No catalog loaded.")
		return
	}
	fmt.Printf("\nAvailable Skills (%d total):\n", len(s.Catalog.Skills))
	for name, sk := range s.Catalog.Skills {
		fmt.Printf("  • %-24s : %s\n", name, sk.Description)
	}
	fmt.Println()
}
