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
				fmt.Printf("\n%s %s\n", BoldYellow("[ACTION] [Remote Steer Received]:"), prompt)
			} else {
				InjectRemotePrompt(prompt)
			}
		}

		rm.OnCancel = func() {
			fmt.Printf("\n%s Remote cancel signal received.\n", BoldRed("[STOP] [Remote Cancel]"))
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

	termWidth := 70
	urlWidth := DisplayCellWidth(sess.PairingURL) + 16
	if termWidth < urlWidth+6 {
		termWidth = urlWidth + 6
	}

	headerTitle := " [REMOTE] mncode Remote Companion "
	titleW := DisplayCellWidth(headerTitle)
	headerDashes := termWidth - titleW - 6
	if headerDashes < 2 {
		headerDashes = 2
	}

	fmt.Println()
	// 1. Top border
	fmt.Printf("%s%s%s%s%s\n",
		BoldCyan("╭──["),
		BoldPastelPink(headerTitle),
		BoldCyan("]"),
		BoldCyan(strings.Repeat("─", headerDashes)),
		BoldCyan("╮"),
	)

	// Helper to print a line with exact padding
	printBoxLine := func(content string) {
		padded := PadToCellWidth(content, termWidth-6)
		fmt.Printf("%s  %s  %s\n", BoldCyan("│"), padded, BoldCyan("│"))
	}

	printBoxLine("")
	printBoxLine(fmt.Sprintf("Session ID:   %s", BoldPastelPink(sess.SessionID)))
	printBoxLine(fmt.Sprintf("Status:       %s", BoldGreen("[ACTIVE] Live Connected & Ready for Mobile")))
	printBoxLine(fmt.Sprintf("Companion:    %s", Bold(sess.PairingURL)))
	printBoxLine("")

	// 2. Middle divider
	fmt.Printf("%s%s%s\n",
		BoldCyan("├"),
		BoldCyan(strings.Repeat("─", termWidth-2)),
		BoldCyan("┤"),
	)

	printBoxLine("")
	printBoxLine("Scan QR with phone camera to steer & answer questions:")
	printBoxLine("")

	if qrCode != "" {
		lines := strings.Split(qrCode, "\n")
		for _, l := range lines {
			trimmed := strings.TrimRight(l, " ")
			if trimmed != "" {
				qrW := DisplayCellWidth(trimmed)
				leftPad := (termWidth - 6 - qrW) / 2
				if leftPad < 0 {
					leftPad = 0
				}
				centeredQR := strings.Repeat(" ", leftPad) + trimmed
				printBoxLine(centeredQR)
			}
		}
	} else {
		printBoxLine(fmt.Sprintf("Open: %s", sess.PairingURL))
	}

	printBoxLine("")
	printBoxLine("[INFO] Tip: Type steer directives or tap choices on mobile screen.")
	printBoxLine("")

	// 3. Bottom border
	fmt.Printf("%s%s%s\n",
		BoldCyan("╰"),
		BoldCyan(strings.Repeat("─", termWidth-2)),
		BoldCyan("╯"),
	)
	fmt.Println()
}
