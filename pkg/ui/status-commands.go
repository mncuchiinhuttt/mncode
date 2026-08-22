package ui

import (
	"fmt"
	"os"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/config"
	"mncode/pkg/stats"

	"golang.org/x/term"
)

// ShowSessionStatus renders the session status as an interactive temporary overlay (press Esc/Enter/q to close)
func ShowSessionStatus(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		renderStatusStatic(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		renderStatusStatic(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 50 {
		width = 80
	}
	cardWidth := width - 2
	if cardWidth > 80 {
		cardWidth = 80
	}

	titleLeft := "SESSION STATUS"
	titleRight := "(Esc / Enter to close)"
	midDashes := cardWidth - 8 - len([]rune(titleLeft)) - len([]rune(titleRight))
	if midDashes < 2 {
		midDashes = 2
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s %s %s %s %s",
		BoldCyan("╭──"), BoldCyan(titleLeft), GrayText(strings.Repeat("─", midDashes)), BoldYellow(titleRight), BoldCyan("──╮")))

	wfName := strings.ToUpper(s.Config.Workflow)
	if wfName == "ULTRA-WORKFLOW" || wfName == "ULTRA WORKFLOW" {
		wfName = "ULTRA WORKFLOW"
	}
	if wfName == "" {
		wfName = "AUTO"
	}

	permLabel := string(s.Config.PermissionMode)
	switch s.Config.PermissionMode {
	case config.PermissionModeBypass:
		permLabel = BoldYellow("BYPASS") + GrayText(" (let him cook with max rizz)")
	case config.PermissionModePlan:
		permLabel = BoldPastelPink("PLAN") + GrayText(" (read-only)")
	case config.PermissionModeAuto:
		permLabel = BoldCyan("AUTO") + GrayText(" (auto-approve)")
	default:
		permLabel = GrayText("ASK (interactive prompt)")
	}

	effortLabel := BoldGreen(strings.ToUpper(s.Config.Effort))
	if s.Config.ThinkingBudget > 0 {
		effortLabel = fmt.Sprintf("%s %s", effortLabel, GrayText(fmt.Sprintf("(%d tokens budget)", s.Config.ThinkingBudget)))
	}

	wsDir := s.WorkspaceDir
	if len([]rune(wsDir)) > cardWidth-20 {
		wsDir = "..." + string([]rune(wsDir)[len([]rune(wsDir))-(cardWidth-23):])
	}

	lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Workspace:"), Bold(wsDir)))
	lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Provider:"), BoldCyan(string(s.Config.Provider))))
	lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Model:"), Bold(s.Config.Model)))
	lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Context Window:"), BoldCyan(fmt.Sprintf("%s (%s tokens)", s.Config.GetContextWindowLabel(), formatNumber(int64(s.Config.GetContextWindowTokens()))))))
	lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Effort Level:"), effortLabel))
	lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Workflow:"), BoldCyan(wfName)))
	lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Permissions:"), permLabel))
	lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("History Size:"), fmt.Sprintf("%d messages", len(s.History))))

	if tracker, ok := s.Tracker.(*stats.Tracker); ok && tracker != nil {
		today := tracker.GetToday()
		lifetime := tracker.GetLifetime()
		lines = append(lines, fmt.Sprintf("  %-16s %s %s", GrayText("Today Tokens:"),
			BoldGreen(formatNumber(today.TotalTokens)), GrayText(fmt.Sprintf("(%d requests)", today.Requests))))
		lines = append(lines, fmt.Sprintf("  %-16s %s %s", GrayText("Total Tokens:"),
			BoldYellow(formatNumber(lifetime.TotalTokens)), GrayText(fmt.Sprintf("(%d requests)", lifetime.Requests))))
	}

	if s.Config.GetTelemetryKey() == "" {
		lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Account:"), GrayText("not linked (run /login)")))
	} else if who, err := fetchWhoAmI(s); err != nil {
		lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Account:"), GrayText("connected (offline cache)")))
	} else {
		label := fmt.Sprintf("%s <%s>", who.User.Name, who.User.Email)
		if who.IsAdmin {
			label += " " + BoldYellow("[admin]")
		}
		lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Account:"), BoldGreen(label)))
		lastSync := s.Config.GetSetting("last_telemetry_sync_date", "")
		if lastSync == "" {
			lastSync = "never (run /sync)"
		}
		lines = append(lines, fmt.Sprintf("  %-16s %s", GrayText("Last Synced:"), GrayText(lastSync)))
	}

	lines = append(lines, GrayText("╰"+strings.Repeat("─", cardWidth-2)+"╯"))

	// Render overlay
	for i, line := range lines {
		if i < len(lines)-1 {
			fmt.Printf("\r\033[K%s\r\n", line)
		} else {
			fmt.Printf("\r\033[K%s", line)
		}
	}
	lastLinesCount := len(lines)

	// Wait for dismiss key
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		b := buf[0]
		switch b {
		case 3, 27, 'q', 'Q', 13, 10, ' ':
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return
		}
	}
}

func renderStatusStatic(s *agent.Session) {
	wfName := strings.ToUpper(s.Config.Workflow)
	if wfName == "ULTRA-WORKFLOW" || wfName == "ULTRA WORKFLOW" {
		wfName = "ULTRA WORKFLOW"
	}
	fmt.Printf("\n%s\n", BoldCyan("Session Status:"))
	fmt.Printf("  Workspace:    %s\n", s.WorkspaceDir)
	fmt.Printf("  Provider:     %s\n", s.Config.Provider)
	fmt.Printf("  Model:        %s\n", s.Config.Model)
	fmt.Printf("  Effort Level: %s (%d tokens budget)\n", BoldGreen(strings.ToUpper(s.Config.Effort)), s.Config.ThinkingBudget)
	fmt.Printf("  Workflow:     %s\n", BoldCyan(wfName))
	fmt.Printf("  Auto-Approve: %v\n", s.Config.AutoApprove)
	fmt.Printf("  History Size: %d messages\n", len(s.History))
	fmt.Println()
}
