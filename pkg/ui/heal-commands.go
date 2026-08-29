package ui

import (
	"context"
	"fmt"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/selfheal"
)

type healTerminalListener struct{}

func (l *healTerminalListener) OnHealStep(step, status string) {
	fmt.Printf("  \033[1;36m>\033[0m \033[1;33m%-22s\033[0m %s\n", step, status)
}

func handleHealCommand(args string, s *agent.Session) {
	if s == nil {
		fmt.Println("\033[31m[Error] Active session required for /heal.\033[0m")
		return
	}

	defect := strings.TrimSpace(args)
	if defect == "" {
		fmt.Print("\n\033[1;36mEnter bug / defect description to reproduce and heal:\033[0m\n> ")
		defect = strings.TrimSpace(readLineRaw())
		if defect == "" {
			fmt.Println("\033[33m[Cancelled] No defect description provided.\033[0m")
			return
		}
	}

	fmt.Println("\n\033[1;36m=== Autonomous Self-Healing Test Harness ===\033[0m")
	fmt.Printf("Defect: %s\n\n", defect)

	listener := &healTerminalListener{}
	coord := selfheal.NewAutoHealCoordinator(s.WorkspaceDir, listener)

	// In interactive CLI, run the fix cycle through agent subagent
	fixPrompt := fmt.Sprintf("Autonomous bug fix: %s. Fix the code and ensure all tests pass.", defect)

	session, err := coord.ExecuteHealingLoop(
		context.Background(),
		defect,
		"go test -count=1 ./...",
		func(ctx context.Context) error {
			runner := &agent.SubagentRunner{ParentSession: s}
			_, runErr := runner.Run(ctx, "debugger", fixPrompt)
			return runErr
		},
	)

	if err != nil {
		fmt.Printf("\n\033[31m[Heal Error] %v\033[0m\n\n", err)
		return
	}

	fmt.Printf("\n\033[1;32m[OK] Successfully reproduced and healed defect (Session: %s)!\033[0m\n\n", session.ID)
}
