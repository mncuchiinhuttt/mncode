package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"

	"golang.org/x/term"
)

type EffortOption struct {
	ID          string
	Label       string
	Budget      int
	Workflow    string
	Description string
}

var effortOptions = []EffortOption{
	{
		ID:          "low",
		Label:       "low",
		Budget:      2048,
		Workflow:    "direct",
		Description: "Fast responses, minimal thinking delay for quick edits",
	},
	{
		ID:          "medium",
		Label:       "medium",
		Budget:      8192,
		Workflow:    "auto",
		Description: "Balanced reasoning for general software engineering",
	},
	{
		ID:          "high",
		Label:       "high",
		Budget:      16384,
		Workflow:    "auto",
		Description: "Deep architectural and coding analysis for complex tasks",
	},
	{
		ID:          "xhigh",
		Label:       "xhigh",
		Budget:      32768,
		Workflow:    "auto",
		Description: "Extended multi-step reasoning for difficult bugs",
	},
	{
		ID:          "max",
		Label:       "max",
		Budget:      64000,
		Workflow:    "auto",
		Description: "Maximum raw reasoning depth for hard algorithms",
	},
	{
		ID:          "pro max",
		Label:       "pro max",
		Budget:      64000,
		Workflow:    "ultra-workflow",
		Description: "Dynamic Workflows (Autonomous Scout -> Plan -> Code -> Test -> Review)",
	},
}

var (
	effortCenters = []int{4, 16, 28, 40, 52, 66}
	effortStarts  = []int{3, 13, 26, 38, 51, 63}
)

// GetEffortOptions exposes the same reasoning catalog used by /effort.
// The returned slice is a copy so desktop clients cannot mutate CLI state.
func GetEffortOptions() []EffortOption {
	return append([]EffortOption(nil), effortOptions...)
}

// OpenInteractiveEffortSlider opens the Claude Code styled Effort spectrum slider
func OpenInteractiveEffortSlider(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		showCurrentEffort(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		showCurrentEffort(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	currentIdx := 2
	for i, opt := range effortOptions {
		if strings.EqualFold(s.Config.Effort, opt.ID) || (s.Config.Workflow == "ultra-workflow" && opt.ID == "pro max") {
			currentIdx = i
			break
		}
	}

	lastLinesCount := 0

	render := func() {
		opt := effortOptions[currentIdx]
		dividerPos := 58
		trackLen := 74

		var lines []string
		lines = append(lines, "", BoldCyan("   Thinking Effort & Reasoning Scale"), "")

		fasterStr := GrayText("Faster")
		smarterStr := GrayText("Smarter")
		spectrumGap := trackLen - len("Faster") - len("Smarter")
		lines = append(lines, fmt.Sprintf("   %s%s%s", fasterStr, strings.Repeat(" ", spectrumGap), smarterStr))

		targetCenter := effortCenters[currentIdx]
		var trackSb strings.Builder
		trackSb.WriteString("   ")
		for i := 0; i < trackLen; i++ {
			if i == targetCenter {
				trackSb.WriteString(BoldPastelPink("▲"))
			} else if i == dividerPos {
				trackSb.WriteString(GrayText("┆"))
			} else {
				trackSb.WriteString(GrayText("─"))
			}
		}
		lines = append(lines, trackSb.String())

		var labelsSb strings.Builder
		labelsSb.WriteString("   ")
		currentCol := 0
		for i, o := range effortOptions {
			pad := effortStarts[i] - currentCol
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
		lines = append(lines, fmt.Sprintf("   %s%s", strings.Repeat(" ", 58), GrayText("xhigh + workflows")), "")

		lines = append(lines, fmt.Sprintf("   %s %s", Bold("Level:      "), BoldPastelPink(strings.ToUpper(opt.Label))))
		lines = append(lines, fmt.Sprintf("   %s %s", GrayText("Description:"), opt.Description))
		lines = append(lines, "", GrayText("   ←/→ to adjust · 1-6 to select · Enter to confirm · Esc to cancel"))

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
			case 'D', 'A':
				if currentIdx > 0 {
					currentIdx--
					render()
				}
				continue
			case 'C', 'B':
				if currentIdx < len(effortOptions)-1 {
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
			chosen := effortOptions[currentIdx]
			s.Config.Effort = chosen.ID
			s.Config.ThinkingBudget = chosen.Budget
			s.Config.Workflow = chosen.Workflow
			_ = config.SaveConfig(s.Config)

			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			if chosen.ID == "pro max" {
				triggerProMaxActivationAnimation()
			} else {
				fmt.Printf("Thinking Effort set to: %s (Workflow: %s)\r\n\r\n",
					BoldGreen(strings.ToUpper(chosen.Label)),
					BoldCyan(strings.ToUpper(s.Config.Workflow)))
			}
			return

		case '1', '2', '3', '4', '5', '6':
			idx := int(b - '1')
			if idx >= 0 && idx < len(effortOptions) {
				currentIdx = idx
				render()
			}
			continue
		}
	}
}
