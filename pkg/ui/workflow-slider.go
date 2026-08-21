package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"

	"golang.org/x/term"
)

type WorkflowOption struct {
	ID          string
	Label       string
	Description string
}

var workflowOptions = []WorkflowOption{
	{
		ID:          "direct",
		Label:       "direct",
		Description: "Single-agent execution without subagent delegation for fast direct answers",
	},
	{
		ID:          "auto",
		Label:       "auto",
		Description: "Adaptive orchestration (automatically spawns subagents when complexity requires)",
	},
	{
		ID:          "plan",
		Label:       "plan",
		Description: "Strict Plan Mode (Read-only codebase + write plans to ./plans/ only)",
	},
	{
		ID:          "plan-first",
		Label:       "plan-first",
		Description: "Mandatory structured planning in ./plans/ before any code modifications",
	},
	{
		ID:          "ultracode",
		Label:       "ultracode",
		Description: "Full 5-phase pipeline: Scout -> Plan -> Code -> Test -> Code Review",
	},
}

var (
	workflowCenters = []int{6, 18, 32, 48, 66}
	workflowStarts  = []int{4, 16, 30, 44, 62}
)

// OpenInteractiveWorkflowSlider opens the Claude Code styled Workflow spectrum slider
func OpenInteractiveWorkflowSlider(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		showCurrentWorkflow(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		showCurrentWorkflow(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	currentIdx := 1 // default to auto
	for i, opt := range workflowOptions {
		if strings.EqualFold(s.Config.Workflow, opt.ID) {
			currentIdx = i
			break
		}
	}

	lastLinesCount := 0

	render := func() {
		opt := workflowOptions[currentIdx]
		trackLen := 74

		var lines []string
		lines = append(lines, "")
		lines = append(lines, BoldCyan("   Agent Workflow & Autonomy Scale"))
		lines = append(lines, "")

		directStr := GrayText("Direct")
		autoStr := GrayText("Autonomous")
		spectrumGap := trackLen - len("Direct") - len("Autonomous")
		lines = append(lines, fmt.Sprintf("   %s%s%s", directStr, strings.Repeat(" ", spectrumGap), autoStr))

		targetCenter := workflowCenters[currentIdx]
		var trackSb strings.Builder
		trackSb.WriteString("   ")
		for i := 0; i < trackLen; i++ {
			if i == targetCenter {
				trackSb.WriteString(BoldPastelPink("▲"))
			} else if i == 56 {
				trackSb.WriteString(GrayText("┆"))
			} else {
				trackSb.WriteString(GrayText("─"))
			}
		}
		lines = append(lines, trackSb.String())

		var labelsSb strings.Builder
		labelsSb.WriteString("   ")
		currentCol := 0
		for i, o := range workflowOptions {
			pad := workflowStarts[i] - currentCol
			if pad > 0 {
				labelsSb.WriteString(strings.Repeat(" ", pad))
				currentCol += pad
			}
			if i == currentIdx {
				labelsSb.WriteString(BoldPastelPink(o.Label))
			} else {
				labelsSb.WriteString(GrayText(o.Label))
			}
			currentCol += len([]rune(o.Label))
		}
		lines = append(lines, labelsSb.String())

		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("   %s %s",
			Bold("Mode:       "), BoldPastelPink(strings.ToUpper(opt.Label))))
		lines = append(lines, fmt.Sprintf("   %s %s", GrayText("Description:"), opt.Description))
		lines = append(lines, "")
		lines = append(lines, GrayText("   ←/→ to adjust · 1-5 to select · Enter to confirm · Esc to cancel"))

		if lastLinesCount > 0 {
			fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
		}

		for i, line := range lines {
			if i < len(lines)-1 {
				fmt.Printf("\r\033[K%s\r\n", line)
			} else {
				fmt.Printf("\r\033[K%s", line)
			}
		}
		lastLinesCount = len(lines)
	}

	render()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'D', 'A': // Left or Up
				if currentIdx > 0 {
					currentIdx--
					render()
				}
				continue
			case 'C', 'B': // Right or Down
				if currentIdx < len(workflowOptions)-1 {
					currentIdx++
					render()
				}
				continue
			}
		}

		b := buf[0]
		switch b {
		case 3, 27, 'q', 'Q':
			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			return

		case 13, 10:
			chosen := workflowOptions[currentIdx]
			s.Config.Workflow = chosen.ID
			_ = config.SaveConfig(s.Config)

			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			fmt.Printf("Workflow Mode set to: %s\r\n\r\n", BoldGreen(strings.ToUpper(chosen.Label)))
			return

		case '1', '2', '3', '4', '5':
			idx := int(b - '1')
			if idx >= 0 && idx < len(workflowOptions) {
				currentIdx = idx
				render()
			}
			continue
		}
	}
}
