package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"strconv"
	"strings"
	"time"
)

// HandleEffortCommand adjusts thinking budget / reasoning effort level
func HandleEffortCommand(parts []string, s *agent.Session) {
	if len(parts) == 1 {
		OpenInteractiveEffortSlider(s)
		return
	}

	arg := strings.ToLower(strings.Join(parts[1:], " "))
	switch arg {
	case "low", "1":
		s.Config.Effort = "low"
		s.Config.ThinkingBudget = 2048
	case "medium", "med", "2":
		s.Config.Effort = "medium"
		s.Config.ThinkingBudget = 8192
	case "high", "3":
		s.Config.Effort = "high"
		s.Config.ThinkingBudget = 16384
	case "xhigh", "extra", "4":
		s.Config.Effort = "xhigh"
		s.Config.ThinkingBudget = 32768
	case "max", "maximum", "5":
		s.Config.Effort = "max"
		s.Config.ThinkingBudget = 64000
	case "pro max", "promax", "pro", "6":
		s.Config.Effort = "pro max"
		s.Config.ThinkingBudget = 64000
		s.Config.Workflow = "ultracode"
	default:
		if num, err := strconv.Atoi(arg); err == nil && num >= 0 {
			s.Config.ThinkingBudget = num
			if num <= 2048 {
				s.Config.Effort = "low"
			} else if num <= 8192 {
				s.Config.Effort = "medium"
			} else if num <= 16384 {
				s.Config.Effort = "high"
			} else if num <= 32768 {
				s.Config.Effort = "xhigh"
			} else {
				s.Config.Effort = "max"
			}
		} else {
			fmt.Printf("Unknown effort '%s'. Use: low, medium, high, xhigh, max, pro max, or token count.\n", arg)
			return
		}
	}

	_ = config.SaveConfig(s.Config)
	RenderStickyHeader(s)

	if s.Config.Effort == "pro max" || s.Config.Workflow == "ultra-workflow" || s.Config.Workflow == "ultracode" {
		triggerProMaxActivationAnimation()
	} else {
		fmt.Printf("Thinking Effort set to: %s (Workflow: %s)\n",
			BoldGreen(strings.ToUpper(s.Config.Effort)),
			BoldCyan(strings.ToUpper(s.Config.Workflow)))
	}
}

func triggerProMaxActivationAnimation() {
	fmt.Println()
	for i := 0; i <= 20; i++ {
		bar := strings.Repeat("█", i) + strings.Repeat("░", 20-i)
		pct := i * 5
		fmt.Printf("\r  \033[1;38;5;212m[ENGAGING MNCODE PRO MAX]\033[0m \033[38;5;218m[%s]\033[0m \033[38;5;225m%d%%\033[0m", bar, pct)
		time.Sleep(15 * time.Millisecond)
	}
	fmt.Printf("\r  \033[1;38;5;218m[MNCODE PRO MAX ACTIVE]\033[0m   \033[1;38;5;212m[████████████████████]\033[0m \033[1;38;5;225mULTRA WORKFLOW\033[0m · \033[1;38;5;219mMax Rizz & Unbounded Cognitive Depth Activated 🔥\033[0m\n\n")
}

func showCurrentEffort(s *agent.Session) {
	fmt.Printf("\n%s\n", BoldCyan("Thinking Effort Configuration:"))
	fmt.Printf("  Current Level:  %s\n", BoldGreen(strings.ToUpper(s.Config.Effort)))
	fmt.Printf("  Workflow Mode:  %s\n", BoldCyan(strings.ToUpper(s.Config.Workflow)))
	fmt.Println()
	fmt.Println("Available levels:")
	fmt.Println("  • [1] low     - Fast responses, minimal thinking")
	fmt.Println("  • [2] medium  - Balanced reasoning for daily tasks")
	fmt.Println("  • [3] high    - Deep architectural and coding analysis")
	fmt.Println("  • [4] xhigh   - Extended multi-step reasoning")
	fmt.Println("  • [5] max     - Maximum raw reasoning depth")
	fmt.Println("  • [6] pro max - Ultra Workflows (Scout -> Plan -> Code -> Test -> Review)")
	fmt.Println()
	fmt.Println(GrayText("  Usage: /effort to open interactive slider, or /effort pro max"))
	fmt.Println()
}

// HandleWorkflowCommand sets or displays active agent workflow mode
func HandleWorkflowCommand(parts []string, s *agent.Session) {
	if len(parts) == 1 {
		OpenInteractiveWorkflowSlider(s)
		return
	}

	mode := strings.ToLower(parts[1])
	switch mode {
	case "auto":
		s.Config.Workflow = "auto"
	case "ultra-workflow", "ultra", "ultracode", "pro":
		s.Config.Workflow = "ultra-workflow"
	case "plan-first":
		s.Config.Workflow = "plan-first"
	case "direct":
		s.Config.Workflow = "direct"
	default:
		fmt.Printf("Unknown workflow '%s'. Use: auto, ultra-workflow, plan-first, direct\n", mode)
		return
	}

	_ = config.SaveConfig(s.Config)
	RenderStickyHeader(s)
	fmt.Printf("Workflow Mode updated to: %s\n", BoldCyan(strings.ToUpper(s.Config.Workflow)))
}

func showCurrentWorkflow(s *agent.Session) {
	fmt.Printf("\n%s\n", BoldCyan("Agent Workflow Modes:"))
	fmt.Printf("  Current Mode: %s\n\n", BoldGreen(strings.ToUpper(s.Config.Workflow)))
	fmt.Println("Available modes:")
	fmt.Println("  • auto           - Autonomous multi-agent orchestration (default)")
	fmt.Println("  • ultra-workflow - Full Multi-Phase Plan & Subagent Pipeline with 2-Pane Split Monitor")
	fmt.Println("  • plan-first     - Mandatory planning in ./plans/ before code execution")
	fmt.Println("  • direct         - Direct single-agent execution without delegation")
	fmt.Println()
	fmt.Println(GrayText("  Usage: /workflow <auto|ultra-workflow|plan-first|direct>"))
	fmt.Println()
}
