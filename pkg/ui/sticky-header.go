package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var (
	currentStickyCleanup func()
	ansiRe               = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
)

func strVisualLen(s string) int {
	clean := ansiRe.ReplaceAllString(s, "")
	return len([]rune(clean))
}

func formatBoxRow(content string, boxWidth int) string {
	vLen := strVisualLen(content)
	pad := boxWidth - 4 - vLen
	if pad < 0 {
		pad = 0
	}
	return fmt.Sprintf("%s %s%s %s",
		GrayText("│"),
		content,
		strings.Repeat(" ", pad),
		GrayText("│"))
}

// EnableStickyHeader configures scrolling region so the Header Card remains permanently pinned at the top
func EnableStickyHeader(s *agent.Session) func() {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return func() {}
	}

	// Clear terminal screen and reset cursor to (1,1)
	fmt.Print("\033[2J\033[H")

	RenderStickyHeader(s)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			RenderStickyHeader(s)
		}
	}()

	cleanup := func() {
		signal.Stop(sigCh)
		fmt.Print("\033[r\033[?25h\r\n")
	}
	currentStickyCleanup = cleanup
	return cleanup
}

// RenderStickyHeader paints the sticky top header card without disrupting active cursor
func RenderStickyHeader(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || height < 15 || width < 60 {
		return
	}

	boxWidth := width - 1
	cardHeight := 9

	// Set scrolling region to rows (cardHeight+1)..height
	fmt.Printf("\033[%d;%dr", cardHeight+1, height)

	isProMax := strings.Contains(strings.ToLower(s.Config.Effort), "pro") ||
		strings.Contains(strings.ToLower(s.Config.Effort), "max") ||
		s.Config.ThinkingBudget >= 64000 ||
		s.Config.Workflow == "ultracode"

	modelName := s.Config.Model
	if modelName == "" {
		modelName = "gemini-3.7-flash-high"
	}
	branch := GetGitBranchOrFolder(s.WorkspaceDir)

	tag := "\033[1;38;5;218mmncode\033[0m \033[1;38;5;212mPRO\033[0m \033[1;38;5;219mMAX\033[0m"
	if !isProMax {
		tag = BoldCyan("mncode")
	}

	titleRight := fmt.Sprintf("%s · %s", modelName, branch)
	dashCount := boxWidth - 10 - strVisualLen(tag) - strVisualLen(titleRight)
	if dashCount < 2 {
		dashCount = 2
	}

	topBorder := fmt.Sprintf("%s %s %s %s %s",
		GrayText("╭──"),
		tag,
		GrayText(strings.Repeat("─", dashCount)),
		GrayText(titleRight),
		GrayText("──╮"))

	var lines []string
	lines = append(lines, topBorder)
	lines = append(lines, formatBoxRow(BoldCyan("  __  __ _  _  ____ ___  ____  ____ "), boxWidth))
	lines = append(lines, formatBoxRow(fmt.Sprintf("%s   %s %s", BoldCyan(" (  \\/  ( \\( )/ ___/ _ \\(    \\(  __)"), Bold("Model:    "), BoldGreen(modelName)), boxWidth))
	lines = append(lines, formatBoxRow(fmt.Sprintf("%s   %s %s (%s)", BoldCyan("  )    ( )  (( (__ )(_) )) D ( ) _) "), Bold("Provider: "), BoldCyan(string(s.Config.Provider)), BoldMagenta(strings.ToUpper(s.Config.Effort))), boxWidth))
	lines = append(lines, formatBoxRow(fmt.Sprintf("%s   %s %s", BoldCyan(" (_/\\/\\_(_)\\_)\\____\\___/(____/(____)"), Bold("Workspace:"), GrayText(s.WorkspaceDir)), boxWidth))
	lines = append(lines, formatBoxRow("", boxWidth))

	accStr := "1 account"
	if s.Accounts != nil && len(s.Accounts.Accounts) > 1 {
		accStr = fmt.Sprintf("%d accounts active", len(s.Accounts.Accounts))
	}
	statsLine := fmt.Sprintf("%s skills · %s agents · %s rules · %s",
		BoldCyan(fmt.Sprintf("%d", len(s.Catalog.Skills))),
		BoldMagenta(fmt.Sprintf("%d", len(s.Catalog.Agents))),
		BoldYellow(fmt.Sprintf("%d", len(s.Catalog.Rules))),
		BoldCyan(accStr))
	lines = append(lines, formatBoxRow(statsLine, boxWidth))

	lines = append(lines, formatBoxRow(GrayText("Type '/' for instant slash menu autocomplete, or enter your message."), boxWidth))
	lines = append(lines, GrayText("╰"+strings.Repeat("─", boxWidth-2)+"╯"))

	// Save cursor position, render header at row 1..9, restore cursor position
	var sb strings.Builder
	sb.WriteString("\0337")
	for i, l := range lines {
		sb.WriteString(fmt.Sprintf("\033[%d;1H\r\033[K%s", i+1, l))
	}
	sb.WriteString("\0338")
	fmt.Print(sb.String())
}

// ResetStickyHeader restores default full-terminal scrolling
func ResetStickyHeader() {
	if currentStickyCleanup != nil {
		currentStickyCleanup()
	} else {
		fmt.Print("\033[r")
	}
}
