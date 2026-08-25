package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/provider"
	"mncode/pkg/skills"
	"os/exec"
	"strings"
)

// HandleSkillCommand handles /skill or /skills command
func HandleSkillCommand(parts []string, s *agent.Session) {
	if len(parts) > 1 {
		sub := strings.ToLower(parts[1])
		if (sub == "install" || sub == "add") && len(parts) > 2 {
			target := parts[2]
			InstallSkillFromSkillsSh(s, target)
			return
		}
		if sub == "market" || sub == "marketplace" {
			PrintCuratedMarketplace()
			return
		}
		if sub == "remove" || sub == "uninstall" && len(parts) > 2 {
			target := parts[2]
			RemoveInstalledSkill(s, target)
			return
		}
		if sub != "list" && sub != "browse" {
			skillQuery := strings.TrimSpace(strings.Join(parts[1:], " "))
			ActivateSkillByName(s, skillQuery)
			return
		}
	}
	OpenInteractiveSkillsBrowser(s)
}

// PrintCuratedMarketplace lists the curated marketplace catalog with install hints.
func PrintCuratedMarketplace() {
	fmt.Printf("\n%s Curated skills (install with %s):\n\n", BoldCyan("🛒 [Marketplace]"), Bold("/skills install <slug>"))
	market, err := skills.GetMarketplace()
	if err != nil {
		fmt.Printf("  %s Could not load the marketplace: %v\n\n", BoldRed("[Error]"), err)
		return
	}
	for _, skill := range market.AvailableSkills {
		status := ""
		if skill.Installed {
			status = fmt.Sprintf("  %s", BoldGreen("installed"))
		}
		fmt.Printf("  %s  %s — %s%s\n", Bold(skill.Slug), Bold(skill.Name), skill.Category, status)
		fmt.Printf("    %s\n", skill.Description)
	}
	fmt.Printf("\n  %s Skills install into ~/.mncode/skills and are shared with the Desktop app.\n\n", GrayText("Tip:"))
}

// RemoveInstalledSkill deletes a user-installed skill from ~/.mncode/skills.
func RemoveInstalledSkill(s *agent.Session, slug string) {
	if err := skills.DeleteInstalledSkill(slug); err != nil {
		fmt.Printf("\n%s %v\n\n", BoldRed("[Error]"), err)
		return
	}
	fmt.Printf("\n%s Removed '%s'.\n\n", BoldGreen("✓"), Bold(slug))
	if reloaded, rErr := skills.LoadCatalog(s.WorkspaceDir); rErr == nil {
		s.Catalog = reloaded
	}
}

// InstallSkillFromSkillsSh installs a skill: curated catalog slugs install
// into ~/.mncode/skills; anything else falls back to the skills.sh CLI.
func InstallSkillFromSkillsSh(s *agent.Session, skillPackage string) {
	if installed, err := skills.InstallMarketplaceSkill(skillPackage); err == nil {
		fmt.Printf("\n%s Installed curated skill '%s' (%s)\n", BoldCyan("📦 [Skill Installer]"), Bold(installed.Name), installed.Category)
		fmt.Printf("  %s Saved to ~/.mncode/skills/%s — shared with the Desktop app.\n\n", BoldGreen("✓"), installed.Slug)
		if reloaded, rErr := skills.LoadCatalog(s.WorkspaceDir); rErr == nil {
			s.Catalog = reloaded
		}
		return
	}
	fmt.Printf("\n%s Installing skill '%s' from skills.sh repository...\n", BoldCyan("📦 [Skill Installer]"), Bold(skillPackage))
	cmd := exec.Command("npx", "-y", "skills", "add", skillPackage)
	cmd.Dir = s.WorkspaceDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  %s Failed to install via skills CLI: %v\n  %s\n\n", BoldRed("[Error]"), err, string(out))
		return
	}
	fmt.Printf("  %s %s\n\n", BoldGreen("✓"), Bold(fmt.Sprintf("Skill '%s' installed successfully!", skillPackage)))
	// Reload catalog
	if reloaded, rErr := skills.LoadCatalog(s.WorkspaceDir); rErr == nil {
		s.Catalog = reloaded
	}
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
