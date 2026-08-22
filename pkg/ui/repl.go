package ui

import (
	"context"
	"fmt"
	"mncode/pkg/agent"
	"strings"
)

// RunREPL starts the interactive command loop with live inline slash completion
func RunREPL(s *agent.Session) {
	// Explicitly disable any mouse grab modes and reset scroll regions
	fmt.Print("\033[?1000l\033[?1002l\033[?1003l\033[?1006l\033[?1015l\033[r\033[?25h")
	fmt.Print("\033[2J\033[H")

	// Check if an update is available and prompt user with arrow keys
	CheckAndPromptStartupUpdate(s)

	printBanner(s)

	if s.Config.GetSetting("copy_on_select", "true") == "true" {
		stopClipboard := StartClipboardWatcher()
		defer stopClipboard()
	}

	StartBackgroundVersionCheck()

	turnIndex := 0
	for {
		input, ok := ReadInlinePrompt(s)
		if !ok {
			break
		}

		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			continue
		}

		if trimmed == "/exit" || trimmed == "/quit" || trimmed == "exit" || trimmed == "quit" {
			PrintRizzGoodbye(s)
			break
		}

		if HandleSlashCommand(trimmed, s) {
			continue
		}

		turnIndex++
		ctx := context.Background()

		resolvedPrompt, attachedBadges := ResolveAtContext(s.WorkspaceDir, trimmed)
		if len(attachedBadges) > 0 {
			fmt.Printf("%s %s\n", BoldCyan("📎 Attached Context:"), BoldPastelPink(strings.Join(attachedBadges, ", ")))
		}

		fmt.Println()
		err := s.ProcessUserInput(ctx, resolvedPrompt)
		if err != nil {
			fmt.Printf("\n%s %v\n", BoldRed("Error:"), err)
		}
		fmt.Println()

		// Process any queued messages sequentially
		for {
			queued := s.DrainMessageQueue()
			if len(queued) == 0 {
				break
			}
			for _, qMsg := range queued {
				turnIndex++
				fmt.Printf("%s %s\n\n", BoldCyan("📥 [Running Queued Prompt]:"), Bold(qMsg))
				qResolved, qBadges := ResolveAtContext(s.WorkspaceDir, qMsg)
				if len(qBadges) > 0 {
					fmt.Printf("%s %s\n", BoldCyan("📎 Attached Context:"), BoldPastelPink(strings.Join(qBadges, ", ")))
				}
				if err := s.ProcessUserInput(ctx, qResolved); err != nil {
					fmt.Printf("\n%s %v\n", BoldRed("Error:"), err)
				}
				fmt.Println()
			}
		}

		CheckAndPrintPeriodicRecap(s, turnIndex)
	}
}

func printBanner(s *agent.Session) {
	PrintHeaderCard(s)
}
