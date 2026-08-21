package ui

import (
	"bufio"
	"fmt"
	"mncode/pkg/accounts"
	"mncode/pkg/agent"
	"os"
	"strings"
	"time"
)

// HandleLoginPrompt handles interactive login for antigravity and codex
func HandleLoginPrompt(parts []string, s *agent.Session) {
	if len(parts) < 2 {
		fmt.Println("Usage: /login <antigravity|codex>")
		return
	}

	target := strings.ToLower(parts[1])
	reader := bufio.NewReader(os.Stdin)

	switch target {
	case "antigravity", "google":
		if len(parts) > 2 && parts[2] == "manual" {
			fmt.Print("Enter Antigravity / Gemini OAuth Token or Bearer Token: ")
			token, _ := reader.ReadString('\n')
			token = strings.TrimSpace(token)
			if token == "" {
				fmt.Println("Login cancelled.")
				return
			}
			fmt.Print("Enter Account Identifier / Email (e.g. user1@gmail.com): ")
			email, _ := reader.ReadString('\n')
			email = strings.TrimSpace(email)
			if email == "" {
				email = fmt.Sprintf("antigravity-%d", time.Now().Unix())
			}
			acc := &accounts.Account{
				ID:          email,
				Email:       email,
				Provider:    accounts.ProviderTypeAntigravity,
				AccessToken: token,
				IsActive:    true,
				CreatedAt:   time.Now(),
			}
			_ = s.Accounts.AddOrUpdate(acc)
			fmt.Printf("%s Antigravity account %s logged in and ready.\n", BoldGreen("[Success]"), Bold(email))
			return
		}

		// Interactive Google OAuth Web Login
		acc, err := accounts.StartAntigravityWebLogin(s.Accounts)
		if err != nil {
			fmt.Printf("%s Google login failed: %v\n", BoldRed("[Error]"), err)
			fmt.Println(GrayText("  Tip: You can also login manually using '/login antigravity manual'"))
			return
		}

		fmt.Printf("\n%s Logged in as %s (%s)\n", BoldGreen("[Success]"), Bold(acc.Email), BoldCyan("Antigravity"))
		fmt.Printf("Account pool updated: %d accounts active!\n\n", len(s.Accounts.Accounts))

	case "codex":
		fmt.Print("Enter Codex / OpenAI API Token: ")
		token, _ := reader.ReadString('\n')
		token = strings.TrimSpace(token)
		if token == "" {
			fmt.Println("Login cancelled.")
			return
		}
		fmt.Print("Enter Account Identifier (e.g. codex-team-1): ")
		id, _ := reader.ReadString('\n')
		id = strings.TrimSpace(id)
		if id == "" {
			id = fmt.Sprintf("codex-%d", time.Now().Unix())
		}
		_, err := accounts.AddCodexAccount(s.Accounts, id, token)
		if err != nil {
			fmt.Printf("%s %v\n", BoldRed("[Error]"), err)
		} else {
			fmt.Printf("%s Codex account %s logged in successfully.\n", BoldGreen("[Success]"), Bold(id))
		}
	default:
		fmt.Printf("Unknown provider '%s'. Supported: antigravity, codex\n", target)
	}
}

func importLocalAccounts(s *agent.Session) {
	imported := 0
	if acc, err := accounts.ImportAntigravityDefaultCreds(s.Accounts); err == nil && acc != nil {
		fmt.Printf("%s Imported local Antigravity credentials: %s\n", BoldGreen("[OK]"), acc.ID)
		imported++
	}
	if acc, err := accounts.ImportCodexCredentials(s.Accounts); err == nil && acc != nil {
		fmt.Printf("%s Imported local Codex/OpenAI credentials: %s\n", BoldGreen("[OK]"), acc.ID)
		imported++
	}
	if imported == 0 {
		fmt.Println("No local credentials found in ~/.gemini or ~/.openai.")
	}
}
