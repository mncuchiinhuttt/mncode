package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"mncode/pkg/agent"
	"mncode/pkg/combos"
)

func runComboCreateWizard(store *combos.Store, session *agent.Session) {
	fmt.Println("\n\033[1;36m═══ Agent Combo Builder Wizard ═══\033[0m")
	fmt.Println("\033[2mCreate a customized multi-agent pipeline, debate swarm, or parallel team.\033[0m")
	fmt.Println()
	fmt.Print("\033[1m1. Combo Name:\033[0m ")
	name := strings.TrimSpace(readLineRaw())
	if name == "" {
		fmt.Println("\033[33m[Cancelled] Name cannot be empty.\033[0m")
		return
	}

	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	fmt.Print("\033[1m2. Description:\033[0m ")
	desc := strings.TrimSpace(readLineRaw())
	if desc == "" {
		desc = fmt.Sprintf("Custom combo swarm for %s", name)
	}

	fmt.Println("\n\033[1m3. Select Execution Mode:\033[0m")
	fmt.Println("  1) Pipeline   - Step-by-step linear data handoff (A -> B -> C)")
	fmt.Println("  2) Debate     - Proposer vs Critic multi-round debate with Decider")
	fmt.Println("  3) Fan-Out    - Concurrent execution on isolated worktrees + Merge")
	fmt.Print("Choose mode [1-3, default 1]: ")
	modeChoice := strings.TrimSpace(readLineRaw())

	mode := combos.ModePipeline
	switch modeChoice {
	case "2", "debate":
		mode = combos.ModeDebate
	case "3", "fan_out", "fanout":
		mode = combos.ModeFanOut
	}

	var members []combos.ComboMember
	fmt.Println("\n\033[1m4. Configure Member Roles:\033[0m")
	fmt.Println("\033[2mSuggested standard roles: planner, architect, advisor, designer, scout, coder, worker, tester, code-reviewer...\033[0m")

	roleIndex := 1
	for {
		fmt.Printf("\n\033[1;33m--- Role #%d ---\033[0m\n", roleIndex)
		fmt.Printf("Role name (e.g. coder, critic, tester, or Enter to finish): ")
		roleName := strings.TrimSpace(readLineRaw())
		if roleName == "" {
			if len(members) == 0 {
				fmt.Println("\033[33mA combo must have at least one role. Please enter a role name.\033[0m")
				continue
			}
			break
		}

		meta := combos.FindRoleMeta(roleName)

		fmt.Printf("Base subagent template [default: %s]: ", meta.DefaultBaseAgent)
		baseAgent := strings.TrimSpace(readLineRaw())
		if baseAgent == "" {
			baseAgent = meta.DefaultBaseAgent
		}

		fmt.Printf("Primary model ['auto' (recommends %s) or specific]: ", meta.AutoPrimaryModel)
		model := strings.TrimSpace(readLineRaw())
		if model == "" {
			model = "auto"
		}

		fmt.Printf("Fallback model ['auto' (recommends %s), 'none', or specific]: ", meta.AutoFallbackModel)
		fallback := strings.TrimSpace(readLineRaw())
		if fallback == "" {
			fallback = "auto"
		}

		fmt.Print("Custom role prompt instructions (optional, press Enter to skip): ")
		promptOverlay := strings.TrimSpace(readLineRaw())

		members = append(members, combos.ComboMember{
			ID:            fmt.Sprintf("m%d", roleIndex),
			Role:          roleName,
			BaseAgent:     baseAgent,
			Model:         model,
			FallbackModel: fallback,
			PromptOverlay: promptOverlay,
		})

		fmt.Printf("\033[32m[OK] Added role '%s' (Model: %s | Fallback: %s)\033[0m\n", roleName, model, fallback)
		roleIndex++
	}

	combo := combos.Combo{
		ID:          slug,
		Name:        name,
		Description: desc,
		Mode:        mode,
		Members:     members,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := store.Save(combo); err != nil {
		fmt.Printf("\033[31m[Error] Failed to save combo: %v\033[0m\n", err)
		return
	}

	fmt.Printf("\n\033[1;32m[SUCCESS] Successfully created Combo '%s' (%s) with %d roles!\033[0m\n", name, slug, len(members))
	fmt.Printf("Run it anytime with: \033[1;36m/combo run %s <your task>\033[0m\n\n", slug)
}

var stdinReader = bufio.NewReader(os.Stdin)

func readLineRaw() string {
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return strings.TrimSpace(line)
	}
	return strings.TrimRight(line, "\r\n")
}
