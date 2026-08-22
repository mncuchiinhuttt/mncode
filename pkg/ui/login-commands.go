package ui

import (
	"fmt"
	"mncode/pkg/accounts"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// HandleLoginPrompt handles interactive login for provider accounts
// (Antigravity/Gemini, Codex/OpenAI) — reached via "/account login <provider>".
// For logging into your mncode web account, see HandleMncodeLoginCommand ("/login").
func HandleLoginPrompt(parts []string, s *agent.Session) {
	if len(parts) < 2 {
		OpenInteractiveLoginMenu(s)
		return
	}

	target := strings.ToLower(parts[1])

	switch target {
	case "antigravity", "google", "gemini":
		if len(parts) > 2 && parts[2] == "manual" {
			token, ok := ReadModalInput("Antigravity Token", "Enter OAuth or Bearer Token:", "")
			if !ok || token == "" {
				fmt.Println("Login cancelled.")
				return
			}
			email, _ := ReadModalInput("Account Identifier", "Enter Account Identifier / Email:", "user1@gmail.com")
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
			fmt.Println(GrayText("  Tip: You can also login manually using '/account login antigravity manual'"))
			return
		}

		fmt.Printf("\n%s Logged in as %s (%s)\n", BoldGreen("[Success]"), Bold(acc.Email), BoldCyan("Antigravity"))
		fmt.Printf("Account pool updated: %d accounts active!\n\n", len(s.Accounts.Accounts))

	case "codex", "openai":
		token, ok := ReadModalInput("Codex / OpenAI Token", "Enter Codex / OpenAI API Token:", "")
		if !ok || token == "" {
			fmt.Println("Login cancelled.")
			return
		}
		id, _ := ReadModalInput("Account Identifier", "Enter Account Identifier (e.g. codex-team-1):", "codex-1")
		if id == "" {
			id = fmt.Sprintf("codex-%d", time.Now().Unix())
		}
		_, err := accounts.AddCodexAccount(s.Accounts, id, token)
		if err != nil {
			fmt.Printf("%s %v\n", BoldRed("[Error]"), err)
		} else {
			fmt.Printf("%s Codex account %s logged in successfully.\n", BoldGreen("[Success]"), Bold(id))
		}

	case "opencode":
		token, ok := ReadModalInput("OpenCode API Key", "Enter OpenCode API Key (sk-...):", "")
		if !ok || token == "" {
			fmt.Println("Login cancelled.")
			return
		}
		id, _ := ReadModalInput("Account Identifier", "Enter Account Identifier (e.g. opencode-main):", "opencode-main")
		if id == "" {
			id = fmt.Sprintf("opencode-%d", time.Now().Unix())
		}
		if s.Accounts != nil {
			_ = s.Accounts.AddOrUpdate(&accounts.Account{
				ID:          id,
				Email:       id,
				Provider:    accounts.ProviderTypeOpenCode,
				AccessToken: token,
				IsActive:    true,
				CreatedAt:   time.Now(),
			})
		}
		s.Config.OpenCodeAPIKey = token
		s.Config.SetSetting("opencode_api_key", token)
		if s.Config.Provider == config.ProviderOpenCode {
			s.Config.APIKey = token
			_ = s.EnsureProvider()
		}
		_ = config.SaveConfig(s.Config)
		fmt.Printf("%s OpenCode API key for %s saved successfully.\n", BoldGreen("[Success]"), Bold(id))

	default:
		fmt.Printf("Unknown provider '%s'. Supported: antigravity, codex, opencode\n", target)
	}
}

// OpenInteractiveLoginMenu displays an arrow-navigable menu for selecting provider to log in
func OpenInteractiveLoginMenu(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Println("Usage: /account login <antigravity|codex|opencode>")
		return
	}

	options := []struct {
		Title       string
		Description string
		Action      func()
	}{
		{
			Title:       "Antigravity / Google Gemini (OAuth Web Login)",
			Description: "Authenticate via Google OAuth browser session or token",
			Action:      func() { HandleLoginPrompt([]string{"/login", "antigravity"}, s) },
		},
		{
			Title:       "Codex / OpenAI (API Key / Token)",
			Description: "Connect OpenAI or Codex API access token",
			Action:      func() { HandleLoginPrompt([]string{"/login", "codex"}, s) },
		},
		{
			Title:       "OpenCode Zen (API Key)",
			Description: "Connect OpenCode API key for free reasoning & specialist models",
			Action:      func() { HandleLoginPrompt([]string{"/login", "opencode"}, s) },
		},
		{
			Title:       "Auto-import local credentials (~/.gemini, ~/.openai)",
			Description: "Scan and import existing local accounts",
			Action:      func() { importLocalAccounts(s) },
		},
	}

	selected := 0
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("Usage: /account login <antigravity|codex>")
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	linesRendered := 0
	render := func() {
		if linesRendered > 0 {
			fmt.Printf("\033[%dA\r\033[J", linesRendered)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s\r\n", BoldPastelPink("╭── [Provider Accounts] ────────────────────────────────────────────────────────╮")))
		sb.WriteString(fmt.Sprintf("│ Select an account provider to connect:%s│\r\n", strings.Repeat(" ", 38)))
		sb.WriteString(fmt.Sprintf("%s\r\n\r\n", BoldPastelPink("╰───────────────────────────────────────────────────────────────────────────────╯")))

		for i, opt := range options {
			if i == selected {
				sb.WriteString(fmt.Sprintf("  %s %s\r\n", BoldCyan("❯"), Bold(opt.Title)))
				sb.WriteString(fmt.Sprintf("    %s\r\n", GrayText(opt.Description)))
			} else {
				sb.WriteString(fmt.Sprintf("    %s\r\n", opt.Title))
				sb.WriteString(fmt.Sprintf("    %s\r\n", GrayText(opt.Description)))
			}
		}
		sb.WriteString(fmt.Sprintf("\r\n  %s\r\n", GrayText("(Use ↑/↓ arrows to navigate, Enter to select, Esc to cancel)")))

		out := sb.String()
		linesRendered = strings.Count(out, "\n")
		fmt.Print(out)
	}

	render()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		if buf[0] == 3 || (buf[0] == 27 && n == 1) || buf[0] == 'q' || buf[0] == 'Q' {
			break
		}
		if buf[0] == 13 || buf[0] == 10 {
			term.Restore(int(os.Stdin.Fd()), oldState)
			if linesRendered > 0 {
				fmt.Printf("\033[%dA\r\033[J", linesRendered)
			}
			options[selected].Action()
			return
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A': // Up
				if selected > 0 {
					selected--
				}
				render()
			case 'B': // Down
				if selected < len(options)-1 {
					selected++
				}
				render()
			}
		} else if n == 1 {
			if buf[0] == 'k' || buf[0] == 'K' {
				if selected > 0 {
					selected--
				}
				render()
			} else if buf[0] == 'j' || buf[0] == 'J' {
				if selected < len(options)-1 {
					selected++
				}
				render()
			} else if buf[0] >= '1' && int(buf[0]-'1') < len(options) {
				selected = int(buf[0] - '1')
				render()
			}
		}
	}

	term.Restore(int(os.Stdin.Fd()), oldState)
	if linesRendered > 0 {
		fmt.Printf("\033[%dA\r\033[J", linesRendered)
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
