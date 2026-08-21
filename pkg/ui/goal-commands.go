package ui

import (
	"bufio"
	"context"
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"
)

// HandleGoalCommand initiates autonomous Goal-Driven execution with live stopwatch
func HandleGoalCommand(parts []string, s *agent.Session) {
	goal := ""
	if len(parts) > 1 {
		goal = strings.TrimSpace(strings.Join(parts[1:], " "))
	} else {
		fmt.Println()
		fmt.Println(BoldPastelPink("  [GOAL-DRIVEN AUTONOMOUS EXECUTION MODE]"))
		fmt.Println(GrayText("  Goal Mode runs persistently with a live stopwatch until the objective is 100% complete."))
		fmt.Println()
		fmt.Print("  Enter your objective or task: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		goal = strings.TrimSpace(input)
		if goal == "" {
			fmt.Println("  Goal execution cancelled.")
			return
		}
	}

	fmt.Println()
	fmt.Printf("%s Starting persistent execution for: %s\n", BoldPastelPink("⏵ [GOAL ACTIVE]"), Bold(goal))
	fmt.Println(GrayText("  Autonomous loop active · Live query stopwatch running..."))
	fmt.Println()

	ctx := context.Background()
	if err := s.ProcessGoal(ctx, goal); err != nil {
		fmt.Printf("\n%s Goal execution encountered error: %v\n", BoldRed("[Error]"), err)
	}
}
