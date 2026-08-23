package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/remote"
)

// HandleRemoteCommand starts or displays the active remote companion session
func HandleRemoteCommand(parts []string, s *agent.Session) {
	serverURL := s.Config.GetSetting("remote_server_url", "")
	if serverURL == "" {
		serverURL = s.Config.GetWebBaseURL()
	}

	apiKey := s.Config.APIKey
	workspaceName := filepath.Base(s.WorkspaceDir)
	if workspaceName == "" || workspaceName == "." {
		workspaceName = "mncode-workspace"
	}

	rm := s.Remote
	if rm == nil || !rm.IsActive {
		rm = remote.NewRemoteManager(serverURL, apiKey)
		ctx := context.Background()

		fmt.Printf("\n%s Connecting to remote bridge server (%s)...\n", BoldCyan("[Remote]"), serverURL)
		sess, err := rm.InitSession(ctx, workspaceName)
		if err != nil {
			fmt.Printf("\n%s Failed to initialize remote companion: %v\n\n", BoldRed("[Error]"), err)
			return
		}

		// Setup event handlers
		rm.OnSteer = func(prompt string) {
			if s.IsExecuting() {
				s.EnqueueSteer(prompt)
				fmt.Printf("\n%s %s\n", BoldYellow("⚡ [Remote Steer Received]:"), prompt)
			} else {
				InjectRemotePrompt(prompt)
			}
		}

		rm.OnCancel = func() {
			fmt.Printf("\n%s Remote cancel signal received.\n", BoldRed("🛑 [Remote Cancel]"))
		}

		s.Remote = rm
		displayRemoteCard(sess)
		return
	}

	// Already active - show existing session card
	if rm.Session != nil {
		displayRemoteCard(rm.Session)
	}
}

func displayRemoteCard(sess *remote.RemoteSession) {
	qrCode := remote.GenerateTerminalQRCode(sess.PairingURL)

	termWidth := 68
	urlWidth := DisplayCellWidth(sess.PairingURL) + 16
	if termWidth < urlWidth+4 {
		termWidth = urlWidth + 4
	}

	borderLine := strings.Repeat("─", termWidth-2)
	headerTitle := " 📱 mncode Remote Companion "
	titleDisplayWidth := DisplayCellWidth(headerTitle)
	headerDashes := termWidth - titleDisplayWidth - 4
	if headerDashes < 2 {
		headerDashes = 2
	}

	fmt.Println()
	fmt.Printf("%s%s%s%s%s\n", BoldCyan("╭──["), BoldPastelPink(headerTitle), BoldCyan("]"), BoldCyan(strings.Repeat("─", headerDashes)), BoldCyan("╮"))

	printBoxLine := func(content string, plainText string) {
		visLen := DisplayCellWidth(plainText)
		pad := termWidth - visLen - 4
		if pad < 0 {
			pad = 0
		}
		fmt.Printf("%s  %s%s  %s\n", BoldCyan("│"), content, strings.Repeat(" ", pad), BoldCyan("│"))
	}

	printBoxLine("", "")
	printBoxLine(
		fmt.Sprintf("Session ID:   %s", BoldPastelPink(sess.SessionID)),
		fmt.Sprintf("Session ID:   %s", sess.SessionID),
	)
	printBoxLine(
		fmt.Sprintf("Status:       %s", BoldGreen("🟢 Live Connected & Ready for Mobile")),
		"Status:       🟢 Live Connected & Ready for Mobile",
	)
	printBoxLine(
		fmt.Sprintf("Companion:    %s", Bold(sess.PairingURL)),
		fmt.Sprintf("Companion:    %s", sess.PairingURL),
	)
	printBoxLine("", "")

	fmt.Printf("%s%s%s\n", BoldCyan("├"), BoldCyan(borderLine), BoldCyan("┤"))
	printBoxLine("", "")
	printBoxLine(
		"Scan QR with phone camera to steer & answer questions:",
		"Scan QR with phone camera to steer & answer questions:",
	)
	printBoxLine("", "")

	if qrCode != "" {
		lines := strings.Split(qrCode, "\n")
		for _, l := range lines {
			trimmed := strings.TrimRight(l, " ")
			if trimmed != "" {
				qrLen := DisplayCellWidth(trimmed)
				leftPad := (termWidth - 4 - qrLen) / 2
				if leftPad < 0 {
					leftPad = 0
				}
				rightPad := termWidth - 4 - qrLen - leftPad
				if rightPad < 0 {
					rightPad = 0
				}
				fmt.Printf("%s  %s%s%s  %s\n", BoldCyan("│"), strings.Repeat(" ", leftPad), trimmed, strings.Repeat(" ", rightPad), BoldCyan("│"))
			}
		}
	} else {
		printBoxLine(fmt.Sprintf("Open: %s", sess.PairingURL), fmt.Sprintf("Open: %s", sess.PairingURL))
	}

	printBoxLine("", "")
	printBoxLine(
		"💡 Tip: Type steer directives or tap choices on mobile screen.",
		"💡 Tip: Type steer directives or tap choices on mobile screen.",
	)
	printBoxLine("", "")
	fmt.Printf("%s%s%s\n", BoldCyan("╰"), BoldCyan(borderLine), BoldCyan("╯"))
	fmt.Println()
}
