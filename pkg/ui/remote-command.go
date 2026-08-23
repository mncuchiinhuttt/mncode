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
	serverURL := s.Config.GetSetting("remote_server_url", "https://mncode.fun")
	if serverURL == "" {
		serverURL = "https://mncode.fun"
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
			s.EnqueueSteer(prompt)
			fmt.Printf("\n%s %s\n", BoldYellow("⚡ [Remote Steer Received]:"), prompt)
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

	fmt.Println()
	fmt.Println(BoldCyan("╭── [ 📱 mncode Remote Companion Active ] ─────────────────────────╮"))
	fmt.Printf("│  Session ID:   %s\n", BoldPastelPink(sess.SessionID))
	fmt.Printf("│  Status:       %s\n", BoldGreen("🟢 Live Connected & Ready for Mobile"))
	fmt.Printf("│  Remote URL:   %s\n", Bold(sess.PairingURL))
	fmt.Println("├──────────────────────────────────────────────────────────────────┤")
	fmt.Println("│  Scan QR Code with your phone camera to control CLI remotely:    │")
	fmt.Println("│                                                                  │")

	if qrCode != "" {
		lines := strings.Split(qrCode, "\n")
		for _, l := range lines {
			if strings.TrimSpace(l) != "" {
				fmt.Printf("│  %s\n", l)
			}
		}
	} else {
		fmt.Printf("│  Open: %s\n", sess.PairingURL)
	}

	fmt.Println("│                                                                  │")
	fmt.Println("│  💡 Tip: You can type live steer directives or answer questions  │")
	fmt.Println("│         from your phone while lying in bed or away from desk!     │")
	fmt.Println(BoldCyan("╰──────────────────────────────────────────────────────────────────╯"))
	fmt.Println()
}
