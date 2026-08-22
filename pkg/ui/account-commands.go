package ui

import (
	"fmt"
	"mncode/pkg/accounts"
	"mncode/pkg/agent"
	"strings"
	"time"
)

// HandleAccountCommand processes /account and /login subcommands
func HandleAccountCommand(parts []string, s *agent.Session) {
	if s.Accounts == nil {
		fmt.Println("Account management is not initialized.")
		return
	}

	if len(parts) == 1 {
		OpenInteractiveAccountMenu(s)
		return
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "login":
		// Reshape ["/account","login","antigravity",...] into what
		// HandleLoginPrompt expects: ["/login","antigravity",...]
		HandleLoginPrompt(append([]string{"/login"}, parts[2:]...), s)
	case "manage", "menu":
		OpenInteractiveAccountMenu(s)
	case "list", "ls":
		showAccounts(s)
	case "switch", "select", "use":
		if len(parts) < 3 {
			fmt.Println("Usage: /account switch <id/email>")
			return
		}
		target := strings.ToLower(parts[2])
		found := false
		for _, acc := range s.Accounts.Accounts {
			if strings.EqualFold(acc.ID, target) || strings.EqualFold(acc.Email, target) {
				for _, other := range s.Accounts.Accounts {
					if other.Provider == acc.Provider {
						other.IsActive = (other.ID == acc.ID)
					}
				}
				_ = s.Accounts.Save()
				fmt.Printf("%s Switched active account to %s (%s)\n", BoldGreen("[Success]"), Bold(acc.Email), acc.Provider)
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("%s Account '%s' not found in pool.\n", BoldRed("[Error]"), target)
		}
	case "add":
		if len(parts) < 5 {
			fmt.Println("Usage: /account add <provider> <id/email> <token>")
			fmt.Println("Example: /account add antigravity my-account-1 ya29.a0...")
			fmt.Println("Example: /account add codex work-account sk-...")
			return
		}
		prov := accounts.AccountProvider(parts[2])
		acc := &accounts.Account{
			ID:          parts[3],
			Email:       parts[3],
			Provider:    prov,
			AccessToken: parts[4],
			IsActive:    true,
			CreatedAt:   time.Now(),
		}
		if err := s.Accounts.AddOrUpdate(acc); err != nil {
			fmt.Printf("%s %v\n", BoldRed("[Error]"), err)
		} else {
			fmt.Printf("%s Added account %s (%s)\n", BoldGreen("[OK]"), Bold(acc.ID), acc.Provider)
		}
	case "remove", "rm", "delete", "logout":
		if len(parts) < 3 {
			fmt.Println("Usage: /account remove <id/email>")
			return
		}
		target := parts[2]
		var toRemove string
		for _, acc := range s.Accounts.Accounts {
			if strings.EqualFold(acc.ID, target) || strings.EqualFold(acc.Email, target) {
				toRemove = acc.ID
				break
			}
		}
		if toRemove == "" {
			toRemove = target
		}
		if err := s.Accounts.Remove(toRemove); err != nil {
			fmt.Printf("%s %v\n", BoldRed("[Error]"), err)
		} else {
			fmt.Printf("%s Removed account %s from pool.\n", BoldGreen("[Success]"), target)
		}
	case "import":
		importLocalAccounts(s)
	case "quota":
		ShowQuotaDashboard(s)
	default:
		fmt.Printf("Unknown subcommand '%s'. Type /help for account commands.\n", sub)
	}
}

func showAccounts(s *agent.Session) {
	if s.Accounts == nil || len(s.Accounts.Accounts) == 0 {
		fmt.Println()
		fmt.Println("No accounts configured in pool. Use '/account login antigravity' to connect an account.")
		fmt.Println()
		return
	}

	fmt.Printf("\n%s (%d accounts configured):\n", BoldCyan("Multi-Account Pool"), len(s.Accounts.Accounts))
	fmt.Printf("  %-32s %-14s %-12s %-8s %s\n", "ID / Email", "Provider", "Status", "Usage", "Last Error")
	fmt.Println("  " + strings.Repeat("─", 74))

	for _, acc := range s.Accounts.Accounts {
		status := BoldGreen("ACTIVE")
		if !acc.IsActive {
			status = GrayText("STANDBY")
		} else if time.Now().Before(acc.CooldownUntil) {
			remaining := time.Until(acc.CooldownUntil).Round(time.Second)
			status = BoldYellow(fmt.Sprintf("COOLDOWN (%s)", remaining))
		}

		fmt.Printf("  %-32s %-14s %-12s %-8d %s\n",
			Bold(acc.ID),
			BoldCyan(string(acc.Provider)),
			status,
			acc.UsageCount,
			GrayText(acc.LastError))
	}
	fmt.Println()
	fmt.Println(GrayText("  Tip: Run '/account' for interactive manager, '/quota' for live quota limits."))
	fmt.Println()
}
