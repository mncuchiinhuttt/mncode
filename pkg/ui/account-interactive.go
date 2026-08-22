package ui

import (
	"fmt"
	"mncode/pkg/accounts"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"sort"

	"golang.org/x/term"
)

// OpenInteractiveAccountMenu opens an interactive CLI account manager
func OpenInteractiveAccountMenu(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		showAccounts(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		showAccounts(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	currentIdx := 0
	lastLinesCount := 0

	for {
		accList := getSortedAccounts(s.Accounts)
		numAccounts := len(accList)
		totalItems := numAccounts + 3

		if currentIdx >= totalItems {
			currentIdx = 0
		}

		var lines []string
		lines = append(lines, "", BoldCyan("   mncode Account Management Pool:"),
			GrayText("   (Use Up/Down or 1-9 to navigate, Enter/Space to select, Esc to exit)"), "")

		for i, acc := range accList {
			prefix := "    "
			emailStr := acc.Email
			if i == currentIdx {
				prefix = BoldPastelPink("  ❯ ")
				emailStr = Bold(acc.Email)
			}
			statusTag := BoldGreen("[ACTIVE]")
			if !acc.IsActive {
				statusTag = GrayText("[STANDBY]")
			}
			lines = append(lines, fmt.Sprintf("%s[%d] %-32s %-12s %s",
				prefix, i+1, emailStr, BoldCyan(string(acc.Provider)), statusTag))
		}

		addGoogleIdx := numAccounts
		quotaIdx := numAccounts + 1
		exitIdx := numAccounts + 2

		gPrefix, qPrefix, ePrefix := "    ", "    ", "    "
		if currentIdx == addGoogleIdx {
			gPrefix = BoldPastelPink("  ❯ ")
		}
		if currentIdx == quotaIdx {
			qPrefix = BoldPastelPink("  ❯ ")
		}
		if currentIdx == exitIdx {
			ePrefix = BoldPastelPink("  ❯ ")
		}

		lines = append(lines,
			fmt.Sprintf("%s[%d] %s", gPrefix, addGoogleIdx+1, BoldYellow("+ Add / Login New Antigravity Account")),
			fmt.Sprintf("%s[%d] %s", qPrefix, quotaIdx+1, BoldCyan("View Quota & Health Dashboard (/quota)")),
			fmt.Sprintf("%s[%d] %s", ePrefix, exitIdx+1, GrayText("Exit Menu")),
			"", fmt.Sprintf("   %s %d accounts in pool", GrayText("Total:"), numAccounts),
			GrayText("   Enter to select · d/x to delete · Esc to exit"),
		)

		if lastLinesCount > 0 {
			fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
		}

		for i, line := range lines {
			if i < len(lines)-1 {
				fmt.Printf("\r\033[K%s\r\n", line)
			} else {
				fmt.Printf("\r\033[K%s", line)
			}
		}
		lastLinesCount = len(lines)

		buf := make([]byte, 3)
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A':
				if currentIdx > 0 {
					currentIdx--
				} else {
					currentIdx = totalItems - 1
				}
				continue
			case 'B':
				if currentIdx < totalItems-1 {
					currentIdx++
				} else {
					currentIdx = 0
				}
				continue
			}
		}

		b := buf[0]
		switch b {
		case 3, 27, 'q', 'Q':
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return

		case 'd', 'D', 'x', 'X':
			if currentIdx < numAccounts {
				targetAcc := accList[currentIdx]
				_ = s.Accounts.Remove(targetAcc.ID)
				if currentIdx > 0 {
					currentIdx--
				}
			}
			continue

		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			choice := int(b - '1')
			if choice < totalItems {
				currentIdx = choice
			}
			continue

		case 13, 10, ' ':
			if currentIdx < numAccounts {
				targetAcc := accList[currentIdx]
				if targetAcc.RefreshToken != "" && targetAcc.Provider == accounts.ProviderTypeAntigravity {
					if freshTok, rErr := accounts.RefreshGoogleToken(targetAcc.RefreshToken, "", ""); rErr == nil && freshTok != "" {
						targetAcc.AccessToken = freshTok
					}
				}
				for _, a := range s.Accounts.Accounts {
					if a.Provider == targetAcc.Provider {
						a.IsActive = (a.ID == targetAcc.ID)
						if a.ID == targetAcc.ID {
							a.AccessToken = targetAcc.AccessToken
						}
					}
				}
				s.Config.APIKey = targetAcc.AccessToken
				if targetAcc.Provider == accounts.ProviderTypeAntigravity {
					s.Config.Provider = config.ProviderAntigravity
					s.Config.BaseURL = ""
				} else if targetAcc.Provider == accounts.ProviderTypeCodex || targetAcc.Provider == accounts.ProviderTypeOpenAI {
					s.Config.Provider = config.ProviderOpenAI
					s.Config.BaseURL = "https://api.openai.com/v1"
				} else if targetAcc.Provider == accounts.ProviderTypeOpenCode {
					s.Config.Provider = config.ProviderOpenCode
					s.Config.BaseURL = "https://opencode.ai/zen/v1"
				}
				s.Provider = nil
				_ = config.SaveConfig(s.Config)
				_ = s.EnsureProvider()

				if lastLinesCount > 0 {
					fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
				}
				fmt.Printf("%s Set %s as primary %s account.\r\n\r\n", BoldGreen("[Success]"), Bold(targetAcc.Email), targetAcc.Provider)
				return

			} else if currentIdx == addGoogleIdx {
				if lastLinesCount > 0 {
					fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
				}
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				HandleLoginPrompt([]string{"/login", "antigravity"}, s)
				return

			} else if currentIdx == quotaIdx {
				if lastLinesCount > 0 {
					fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
				}
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				ShowQuotaDashboard(s)
				return

			} else {
				if lastLinesCount > 0 {
					fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
				}
				return
			}
		}
	}
}

func getSortedAccounts(store *accounts.Store) []*accounts.Account {
	if store == nil {
		return nil
	}
	var list []*accounts.Account
	for _, acc := range store.Accounts {
		list = append(list, acc)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].IsActive != list[j].IsActive {
			return list[i].IsActive
		}
		return list[i].Email < list[j].Email
	})
	return list
}
