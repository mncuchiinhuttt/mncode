package ui

import (
	"context"
	"fmt"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/combos"
)

func handleComboCommand(args string, session *agent.Session) {
	if session == nil {
		fmt.Println("\033[31m[Error] Active session required for combo operations.\033[0m")
		return
	}

	store, err := combos.NewStore(session.WorkspaceDir)
	if err != nil {
		fmt.Printf("\033[31m[Error] Could not initialize combo store: %v\033[0m\n", err)
		return
	}

	parts := strings.Fields(strings.TrimSpace(args))
	subcmd := ""
	if len(parts) > 0 {
		subcmd = strings.ToLower(parts[0])
	}

	switch subcmd {
	case "", "list":
		renderComboList(store)
	case "create", "new":
		runComboCreateWizard(store, session)
	case "show", "inspect":
		if len(parts) < 2 {
			fmt.Println("\033[33mUsage: /combo show <combo-id>\033[0m")
			return
		}
		renderComboDetails(store, parts[1])
	case "delete", "rm":
		if len(parts) < 2 {
			fmt.Println("\033[33mUsage: /combo delete <combo-id>\033[0m")
			return
		}
		if err := store.Delete(parts[1]); err != nil {
			fmt.Printf("\033[31m[Error] %v\033[0m\n", err)
		} else {
			fmt.Printf("\033[32m[OK] Deleted combo %q\033[0m\n", parts[1])
		}
	case "run":
		if len(parts) < 2 {
			fmt.Println("\033[33mUsage: /combo run <combo-id> [optional prompt]\033[0m")
			return
		}
		comboID := parts[1]
		prompt := ""
		if len(parts) > 2 {
			prompt = strings.Join(parts[2:], " ")
		}
		runComboInteractive(store, session, comboID, prompt)
	default:
		// If user typed /combo <name> [prompt] directly
		runComboInteractive(store, session, subcmd, strings.Join(parts[1:], " "))
	}
}

func renderComboList(store *combos.Store) {
	list := store.List()
	fmt.Println("\n\033[1;36m═══ mncode Agent Combos & Swarms ═══\033[0m")
	fmt.Printf("\033[2m%-18s %-10s %-8s %s\033[0m\n", "COMBO ID", "MODE", "ROLES", "DESCRIPTION")
	fmt.Println(strings.Repeat("─", 72))

	for _, c := range list {
		badge := ""
		if c.IsBuiltin {
			badge = "\033[35m[Preset]\033[0m"
		} else {
			badge = "\033[32m[Custom]\033[0m"
		}
		var roles []string
		for _, m := range c.Members {
			roles = append(roles, m.Role)
		}
		rolesStr := strings.Join(roles, " ➔ ")
		if len(rolesStr) > 28 {
			rolesStr = rolesStr[:25] + "..."
		}
		fmt.Printf("%-18s %-10s %-8d %s %s\n", c.ID, string(c.Mode), len(c.Members), badge, c.Description)
		fmt.Printf("   \033[2mRoles: %s\033[0m\n\n", strings.Join(roles, " ➔ "))
	}
	fmt.Println("\033[2mUse '/combo run <id>' to execute or '/combo create' to build a custom swarm.\033[0m")
}

func renderComboDetails(store *combos.Store, id string) {
	c, ok := store.Get(id)
	if !ok {
		fmt.Printf("\033[31m[Error] Combo %q not found.\033[0m\n", id)
		return
	}
	fmt.Printf("\n\033[1;36mCombo: %s (%s)\033[0m\n", c.Name, c.ID)
	fmt.Printf("\033[2mMode: %s | Description: %s\033[0m\n\n", c.Mode, c.Description)
	fmt.Println("Members & Role Roster:")
	for i, m := range c.Members {
		pModel, fbModel := combos.ResolveRoleModels(m)
		fbStr := "none"
		if fbModel != "" {
			fbStr = fbModel
		}
		fmt.Printf("  %d. \033[1;33m%-14s\033[0m (Agent: %-10s | Model: \033[32m%s\033[0m | Fallback: \033[36m%s\033[0m)\n", i+1, m.Role, m.BaseAgent, pModel, fbStr)
		if m.PromptOverlay != "" {
			fmt.Printf("     \033[2mInstructions: %s\033[0m\n", m.PromptOverlay)
		}
	}
	fmt.Println()
}

func runComboInteractive(store *combos.Store, session *agent.Session, comboID, prompt string) {
	c, ok := store.Get(comboID)
	if !ok {
		fmt.Printf("\033[31m[Error] Combo %q not found. Type '/combo list' to view available swarms.\033[0m\n", comboID)
		return
	}
	if prompt == "" {
		fmt.Printf("\033[1;36mEnter task for Combo '%s' (%s):\033[0m ", c.Name, c.ID)
		prompt = readLineRaw()
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		fmt.Println("\033[33m[Cancelled] No prompt provided.\033[0m")
		return
	}

	exec := newSessionComboExecutor(session)
	hud := newTerminalComboHUD()
	runner := combos.NewRunner(store, exec, hud)

	fmt.Printf("\n\033[1;35m🚀 Launching Combo Swarm: %s [%s]\033[0m\n\n", c.Name, c.Mode)
	res, err := runner.Run(context.Background(), comboID, prompt)
	if err != nil {
		fmt.Printf("\n\033[31m[Combo Error] %v\033[0m\n\n", err)
		return
	}

	fmt.Println("\n\033[1;32m═══ Final Deliverable ═══\033[0m")
	fmt.Println(res.FinalOutput)
	fmt.Println()
}
