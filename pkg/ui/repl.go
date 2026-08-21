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

	stopClipboard := StartClipboardWatcher()
	defer stopClipboard()

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
			fmt.Println("Goodbye!")
			break
		}

		if HandleSlashCommand(trimmed, s) {
			continue
		}

		turnIndex++
		ctx := context.Background()
		fmt.Println()
		err := s.ProcessUserInput(ctx, trimmed)
		if err != nil {
			fmt.Printf("\n%s %v\n", BoldRed("Error:"), err)
		}
		fmt.Println()

		CheckAndPrintPeriodicRecap(s, turnIndex)
	}
}

func printBanner(s *agent.Session) {
	PrintHeaderCard(s)
}
