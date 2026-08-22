package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"strconv"
	"strings"
)

// HandleUndoCommand reverts the latest turn and restores files
func HandleUndoCommand(parts []string, s *agent.Session) {
	msg, err := s.RollbackLastTurn()
	if err != nil {
		fmt.Printf("\n%s %v\n\n", BoldRed("[Undo Error]"), err)
		return
	}
	fmt.Println()
	fmt.Printf("  %s %s\n", BoldGreen("✓"), Bold(msg))
	fmt.Printf("  %s %s\n", GrayText("Status:"), GrayText("Workspace files restored and last turn removed from active memory."))
	fmt.Println()
}

// HandleRewindCommand rewinds N turns or presents previous checkpoint list
func HandleRewindCommand(parts []string, s *agent.Session) {
	turns := 1
	if len(parts) > 1 {
		if val, err := strconv.Atoi(parts[1]); err == nil && val > 0 {
			turns = val
		}
	}

	fmt.Printf("\n%s Rewinding %d turn(s)...\n", BoldPastelPink("⏪ [Rewind]"), turns)
	for i := 0; i < turns; i++ {
		_, err := s.RollbackLastTurn()
		if err != nil {
			fmt.Printf("  %s Stopped at step %d: %v\n", BoldYellow("!"), i+1, err)
			break
		}
	}
	fmt.Printf("  %s %s\n\n", BoldGreen("✓"), Bold(fmt.Sprintf("Rewound %d turn(s) successfully.", turns)))
}

// HandleCheckpointCommand manages manual and automatic snapshots
func HandleCheckpointCommand(parts []string, s *agent.Session) {
	if len(parts) > 1 {
		sub := strings.ToLower(parts[1])
		switch sub {
		case "list", "ls":
			list, _ := s.ListCheckpoints()
			if len(list) == 0 {
				fmt.Printf("\n%s No checkpoints found in .mncode/checkpoints/\n\n", GrayText("[Checkpoints]"))
				return
			}
			fmt.Printf("\n%s (%d snapshots):\n", BoldCyan("📸 Saved Checkpoints"), len(list))
			for _, cp := range list {
				fmt.Printf("  • %-20s %s (%s)\n",
					Bold(cp.ID),
					Colorize(GetCurrentTheme().Text, cp.Summary),
					GrayText(cp.Timestamp.Format("15:04:05")))
			}
			fmt.Println()
			return

		case "create", "save":
			summary := "Manual checkpoint"
			if len(parts) > 2 {
				summary = strings.Join(parts[2:], " ")
			}
			cp, err := s.CreateTurnCheckpoint(len(s.History), summary)
			if err != nil {
				fmt.Printf("\n%s %v\n\n", BoldRed("[Error]"), err)
				return
			}
			fmt.Printf("\n%s Checkpoint created: %s (%s)\n\n", BoldGreen("✓"), Bold(cp.ID), summary)
			return
		}
	}

	fmt.Println()
	fmt.Println(BoldCyan("CHECKPOINT COMMANDS:"))
	fmt.Println("  /checkpoint create <name>  - Create a manual snapshot of workspace")
	fmt.Println("  /checkpoint list           - List all turn checkpoints")
	fmt.Println("  /undo                      - Revert last agent turn and file edits")
	fmt.Println("  /rewind <N>                - Rewind N previous turns")
	fmt.Println()
}
