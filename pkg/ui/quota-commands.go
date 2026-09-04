package ui

import (
	"fmt"
	"mncode/pkg/accounts"
	"mncode/pkg/agent"
	"strings"
)

// ShowQuotaDashboard probes and displays the quota and health status of all accounts
func ShowQuotaDashboard(s *agent.Session) {
	if s.Accounts == nil || len(s.Accounts.Accounts) == 0 {
		fmt.Println()
		fmt.Println(BoldYellow("No accounts registered in pool."))
		fmt.Println(GrayText("Run '/login antigravity' to authenticate with Google, or '/account add' to connect."))
		fmt.Println()
		return
	}

	fmt.Println()
	fmt.Println(BoldCyan("┌─────────────────────────────────────────────────────────────────────────────┐"))
	fmt.Printf("%s  %s%s%s\n", BoldCyan("│"), Bold("ACCOUNT & QUOTA HEALTH DASHBOARD"), strings.Repeat(" ", 43), BoldCyan("│"))
	fmt.Println(BoldCyan("└─────────────────────────────────────────────────────────────────────────────┘"))
	fmt.Println()

	healthyCount := 0
	activeAccountEmail := ""

	for _, acc := range s.Accounts.Accounts {
		info := accounts.CheckAccountQuota(s.Accounts, acc)
		if info.IsHealthy {
			healthyCount++
		}
		if acc.IsActive {
			activeAccountEmail = info.AccountID
		}

		dot := BoldGreen("●")
		statusTag := BoldGreen("[ACTIVE]")
		if !acc.IsActive {
			dot = GrayText("○")
			statusTag = GrayText("[STANDBY]")
		}
		if !info.IsHealthy {
			dot = BoldRed("[FAIL]")
			statusTag = BoldRed("[" + info.Status + "]")
		}

		provTag := BoldCyan(string(info.Provider))
		latTag := BoldGreen(fmt.Sprintf("%dms", info.LatencyMs))

		linePrefix := fmt.Sprintf(" %s %s %s ", dot, Bold(info.AccountID), statusTag)
		rightSide := fmt.Sprintf(" %s (%s)", provTag, latTag)
		fillLen := 78 - len(stripAnsi(linePrefix)) - len(stripAnsi(rightSide))
		if fillLen < 3 {
			fillLen = 3
		}
		divider := strings.Repeat("─", fillLen)

		fmt.Printf("%s%s%s\n", linePrefix, GrayText(divider), rightSide)

		if info.Tier != "" {
			fmt.Printf("   Subscription : %s\n", Bold(info.Tier))
		}
		if info.ProjectID != "" {
			fmt.Printf("   Project ID   : %s\n", GrayText(info.ProjectID))
		}
		if info.ExpiresInStr != "" {
			fmt.Printf("   Token Expiry : %s\n", BoldCyan(info.ExpiresInStr))
		}

		if len(info.ModelQuotas) > 0 {
			fmt.Println(Bold("   Model Quotas :"))
			for _, mq := range info.ModelQuotas {
				bar := renderProgressBar(mq.RemainingPercentage, 20)
				resetStr := ""
				if mq.ResetInStr != "" {
					resetStr = fmt.Sprintf(" %s", GrayText(fmt.Sprintf("(%s)", mq.ResetInStr)))
				}
				fmt.Printf("     %-28s %s%s\n", mq.DisplayName, bar, resetStr)
			}
		} else if len(info.AvailableModels) > 0 {
			modelsStr := strings.Join(info.AvailableModels, ", ")
			if len(modelsStr) > 65 {
				modelsStr = modelsStr[:65] + "..."
			}
			fmt.Printf("   Models       : %s\n", GrayText(modelsStr))
		}

		if info.MaxContext > 0 || info.RPMRemaining != "" {
			fmt.Printf("   Limits       : %s tokens context | %s | %s\n",
				formatNumber(int64(info.MaxContext)),
				info.RPMRemaining,
				info.TPMRemaining)
		}

		if info.ErrorMessage != "" {
			fmt.Printf("   Notice       : %s\n", BoldYellow(info.ErrorMessage))
		}
		fmt.Println()
	}

	fmt.Println(GrayText("─────────────────────────────────────────────────────────────────────────────"))
	fmt.Printf(" Pool Summary : %s/%s accounts active & healthy | Primary: %s\n",
		BoldGreen(fmt.Sprintf("%d", healthyCount)),
		Bold(fmt.Sprintf("%d", len(s.Accounts.Accounts))),
		BoldCyan(activeAccountEmail))
	fmt.Println(GrayText(" Management   : Type '/account' to manage, switch active, or remove accounts."))
	fmt.Println()
}

func renderProgressBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int((pct / 100.0) * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	if pct >= 80 {
		return fmt.Sprintf("%s %3.0f%%", BoldGreen("["+bar+"]"), pct)
	} else if pct >= 40 {
		return fmt.Sprintf("%s %3.0f%%", "\033[1;38;5;218m["+bar+"]\033[0m", pct)
	} else if pct >= 20 {
		return fmt.Sprintf("%s %3.0f%%", BoldYellow("["+bar+"]"), pct)
	}
	return fmt.Sprintf("%s %3.0f%%", BoldRed("["+bar+"]"), pct)
}

func stripAnsi(str string) string {
	var sb strings.Builder
	inEsc := false
	for i := 0; i < len(str); i++ {
		if str[i] == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if str[i] == 'm' {
				inEsc = false
			}
			continue
		}
		sb.WriteByte(str[i])
	}
	return sb.String()
}
