package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

var ansiStripRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func visualLen(s string) int {
	clean := ansiStripRe.ReplaceAllString(s, "")
	return len([]rune(clean))
}

func printBoxLine(content string, cardWidth int) {
	vLen := visualLen(content)
	pad := cardWidth - 4 - vLen
	if pad < 0 {
		pad = 0
	}
	fmt.Printf("%s %s%s %s\n",
		GrayText("│"),
		content,
		strings.Repeat(" ", pad),
		GrayText("│"))
}

// PrintHeaderCard renders a standalone, framed header box displaying active environment and stats
func PrintHeaderCard(s *agent.Session) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		width = 80
	}
	cardWidth := width - 2

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
	tagLen := 14
	if !isProMax {
		tag = BoldCyan("mncode")
		tagLen = 6
	}

	titleRight := fmt.Sprintf("%s · %s", modelName, branch)
	dashCount := cardWidth - 10 - tagLen - len([]rune(titleRight))
	if dashCount < 2 {
		dashCount = 2
	}

	topBorder := fmt.Sprintf("%s %s %s %s %s",
		GrayText("╭──"),
		tag,
		GrayText(strings.Repeat("─", dashCount)),
		GrayText(titleRight),
		GrayText("──╮"))

	fmt.Println()
	fmt.Println(topBorder)
	printBoxLine(BoldCyan("  __  __ _  _  ____ ___  ____  ____ "), cardWidth)
	printBoxLine(fmt.Sprintf("%s   %s %s", BoldCyan(" (  \\/  ( \\( )/ ___/ _ \\(    \\(  __)"), Bold("Model:    "), BoldGreen(modelName)), cardWidth)
	printBoxLine(fmt.Sprintf("%s   %s %s (%s)", BoldCyan("  )    ( )  (( (__ )(_) )) D ( ) _) "), Bold("Provider: "), BoldCyan(string(s.Config.Provider)), BoldMagenta(strings.ToUpper(s.Config.Effort))), cardWidth)
	printBoxLine(fmt.Sprintf("%s   %s %s", BoldCyan(" (_/\\/\\_(_)\\_)\\____\\___/(____/(____)"), Bold("Workspace:"), GrayText(s.WorkspaceDir)), cardWidth)
	printBoxLine("", cardWidth)

	if s.Catalog != nil {
		accStr := "1 account"
		if s.Accounts != nil && len(s.Accounts.Accounts) > 1 {
			accStr = fmt.Sprintf("%d accounts active", len(s.Accounts.Accounts))
		}
		statsLine := fmt.Sprintf("%s skills · %s agents · %s rules · %s",
			BoldCyan(fmt.Sprintf("%d", len(s.Catalog.Skills))),
			BoldMagenta(fmt.Sprintf("%d", len(s.Catalog.Agents))),
			BoldYellow(fmt.Sprintf("%d", len(s.Catalog.Rules))),
			BoldCyan(accStr))
		printBoxLine(statsLine, cardWidth)
	}

	printBoxLine("", cardWidth)
	printBoxLine(GrayText("Type '/' for instant slash menu autocomplete, or enter your message."), cardWidth)
	fmt.Println(GrayText("╰" + strings.Repeat("─", cardWidth-2) + "╯"))
	fmt.Println()
}
