package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"

	"golang.org/x/term"
)

type ContextWindowOption struct {
	ID          string
	Label       string
	Tokens      int
	Description string
}

var contextWindowOptions = []ContextWindowOption{
	{ID: "200k", Label: "200K", Tokens: 200000, Description: "Standard Claude/OpenCode context window (200k tokens)"},
	{ID: "300k", Label: "300K", Tokens: 300000, Description: "Extended workspace context window (300k tokens)"},
	{ID: "500k", Label: "500K", Tokens: 500000, Description: "Massive code & documents context window (500k tokens)"},
	{ID: "1m", Label: "1M", Tokens: 1000000, Description: "Gemini / Antigravity ultra 1,000,000 tokens cognitive depth"},
}

var (
	cwCenters = []int{7, 25, 45, 65}
	cwStarts  = []int{5, 23, 43, 64}
)

// OpenInteractiveContextWindowSlider opens the Context Window Size slider
func OpenInteractiveContextWindowSlider(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		showCurrentContextWindow(s)
		return
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		showCurrentContextWindow(s)
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	currentIdx := 0
	currLabel := strings.ToUpper(s.Config.GetContextWindowLabel())
	for i, opt := range contextWindowOptions {
		if strings.EqualFold(opt.Label, currLabel) || strings.EqualFold(opt.ID, s.Config.ContextWindow) {
			currentIdx = i
			break
		}
	}

	lastLinesCount := 0

	render := func() {
		opt := contextWindowOptions[currentIdx]
		trackLen := 74

		var lines []string
		lines = append(lines, "", BoldCyan("   Context Window Size & Token Capacity"), "")

		standardStr := GrayText("Compact")
		massiveStr := GrayText("Massive (1M)")
		spectrumGap := trackLen - len("Compact") - len("Massive (1M)")
		lines = append(lines, fmt.Sprintf("   %s%s%s", standardStr, strings.Repeat(" ", spectrumGap), massiveStr))

		targetCenter := cwCenters[currentIdx]
		var trackSb strings.Builder
		trackSb.WriteString("   ")
		for i := 0; i < trackLen; i++ {
			if i == targetCenter {
				trackSb.WriteString(BoldPastelPink("▲"))
			} else if i == 55 {
				trackSb.WriteString(GrayText("┆"))
			} else {
				trackSb.WriteString(GrayText("─"))
			}
		}
		lines = append(lines, trackSb.String())

		var labelsSb strings.Builder
		labelsSb.WriteString("   ")
		currentCol := 0
		for i, o := range contextWindowOptions {
			pad := cwStarts[i] - currentCol
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
		lines = append(lines, fmt.Sprintf("   %s %s (%s tokens)",
			Bold("Context Window:"), BoldPastelPink(opt.Label), BoldCyan(fmt.Sprintf("%d", opt.Tokens))))
		lines = append(lines, fmt.Sprintf("   %s %s", GrayText("Description:   "), opt.Description))
		lines = append(lines, "")
		lines = append(lines, GrayText("   ←/→ to adjust · 1-4 to select · Enter to confirm · Esc to cancel"))

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
				if currentIdx < len(contextWindowOptions)-1 {
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
			chosen := contextWindowOptions[currentIdx]
			s.Config.ContextWindow = chosen.ID
			s.Config.SetSetting("context_window", chosen.ID)
			_ = config.SaveConfig(s.Config)

			if lastLinesCount > 0 {
				fmt.Printf("\r\033[%dA\033[J", lastLinesCount-1)
			}
			fmt.Printf("Context Window Size updated to: %s (%s tokens)\r\n\r\n",
				BoldGreen(chosen.Label), BoldCyan(fmt.Sprintf("%d", chosen.Tokens)))
			return

		case '1', '2', '3', '4':
			idx := int(b - '1')
			if idx >= 0 && idx < len(contextWindowOptions) {
				currentIdx = idx
				render()
			}
			continue
		}
	}
}

func showCurrentContextWindow(s *agent.Session) {
	fmt.Printf("\n%s %s (%s tokens)\n\n",
		BoldCyan("Current Context Window:"),
		BoldGreen(s.Config.GetContextWindowLabel()),
		BoldCyan(fmt.Sprintf("%d", s.Config.GetContextWindowTokens())))
}
