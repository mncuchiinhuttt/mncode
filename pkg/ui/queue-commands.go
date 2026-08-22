package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"strings"
)

// HandleSteerCommand injects real-time steering directives into the active agent loop
func HandleSteerCommand(parts []string, s *agent.Session) {
	if len(parts) < 2 {
		fmt.Println()
		fmt.Println(BoldCyan("STEER COMMAND USAGE:"))
		fmt.Println("  /steer <guidance>        - Inject high-priority steering into ongoing agent thought")
		fmt.Println("  /steer list              - View pending steering directives")
		fmt.Println("  /steer clear             - Clear pending steer queue")
		fmt.Println()
		fmt.Println(GrayText("Example: /steer stop creating new files, edit existing files instead"))
		fmt.Println()
		return
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "list", "ls":
		s.QueueMu.Lock()
		steers := s.SteerQueue
		s.QueueMu.Unlock()
		if len(steers) == 0 {
			fmt.Printf("\n%s No pending steer directives.\n\n", GrayText("[Steer Queue]"))
			return
		}
		fmt.Printf("\n%s (%d items):\n", BoldPastelPink("🎯 Pending Steer Directives"), len(steers))
		for i, st := range steers {
			fmt.Printf("  %d. %s\n", i+1, Colorize(GetCurrentTheme().Text, st))
		}
		fmt.Println()
	case "clear", "reset":
		_ = s.DrainSteer()
		fmt.Printf("\n%s Steer queue cleared.\n\n", BoldGreen("[Success]"))
	default:
		guidance := strings.TrimSpace(strings.Join(parts[1:], " "))
		s.EnqueueSteer(guidance)
		fmt.Printf("\n%s \"%s\"\n%s\n\n",
			BoldPastelPink("🎯 [Steer Directive Enqueued]"),
			Bold(guidance),
			GrayText("Will be injected with top priority into the agent's upcoming thought loop."))
	}
}

// HandleQueueCommand enqueues messages for automatic execution in sequence
func HandleQueueCommand(parts []string, s *agent.Session) {
	if len(parts) < 2 {
		fmt.Println()
		fmt.Println(BoldCyan("MESSAGE QUEUE USAGE:"))
		fmt.Println("  /queue <prompt>          - Enqueue a prompt to execute immediately after current task")
		fmt.Println("  /queue list              - View pending message queue")
		fmt.Println("  /queue clear             - Clear all queued messages")
		fmt.Println()
		fmt.Println(GrayText("Example: /queue run tests and check if everything passes"))
		fmt.Println()
		return
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "list", "ls":
		s.QueueMu.Lock()
		msgs := s.MessageQueue
		s.QueueMu.Unlock()
		if len(msgs) == 0 {
			fmt.Printf("\n%s Message queue is currently empty.\n\n", GrayText("[Message Queue]"))
			return
		}
		fmt.Printf("\n%s (%d messages):\n", BoldCyan("📥 Enqueued Messages"), len(msgs))
		for i, m := range msgs {
			fmt.Printf("  %d. %s\n", i+1, Colorize(GetCurrentTheme().Text, m))
		}
		fmt.Println()
	case "clear", "reset":
		_ = s.DrainMessageQueue()
		fmt.Printf("\n%s Message queue cleared.\n\n", BoldGreen("[Success]"))
	default:
		prompt := strings.TrimSpace(strings.Join(parts[1:], " "))
		s.EnqueueMessage(prompt)
		fmt.Printf("\n%s \"%s\"\n%s\n\n",
			BoldCyan("📥 [Message Enqueued]"),
			Bold(prompt),
			GrayText("Will run automatically right after the active turn completes."))
	}
}
