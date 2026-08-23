package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

func printBoxLine(content string, cardWidth int) {
	inner := PadToCellWidth(content, cardWidth-4)
	fmt.Printf("%s %s %s\n",
		GrayText("│"),
		inner,
		GrayText("│"))
}

// PrintHeaderCard renders a standalone, framed header box displaying active environment and stats
func PrintHeaderCard(s *agent.Session) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		width = 80
	}
	cardWidth := width - 2

	isProMax := s.Config.Workflow == "ultra-workflow" ||
		strings.EqualFold(s.Config.Effort, "pro max") ||
		strings.EqualFold(s.Config.Effort, "promax")

	modelName := s.Config.Model
	if modelName == "" {
		modelName = "gemini-3.7-flash-high"
	}
	branch := GetGitBranchOrFolder(s.WorkspaceDir)

	tag := "\033[1;38;5;218mmncode\033[0m \033[1;38;5;212mPRO\033[0m \033[1;38;5;219mMAX\033[0m"
	if !isProMax {
		tag = BoldCyan("mncode")
	}
	tagWidth := DisplayCellWidth(tag)

	titleRight := fmt.Sprintf("%s · %s", modelName, branch)
	titleRightWidth := DisplayCellWidth(titleRight)

	// Top border structure: "╭── " (4) + tag + " " (1) + dashes + " " (1) + titleRight + " ──╮" (4) -> total fixed overhead = 10
	dashCount := cardWidth - 10 - tagWidth - titleRightWidth
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

	if s.Config.GetSetting("show_tips", "true") == "true" {
		tips := []string{
			"Use /diff to view uncommitted changes in your workspace.",
			"Use /btw <question> to ask quick side questions without polluting task history.",
			"Type /steer <guidance> to guide ongoing agent reasoning with top priority.",
			"Type /queue <prompt> to enqueue upcoming tasks without waiting for current turn.",
			"Type /status (dismiss with Esc) to inspect live session metadata and tokens.",
			"Type /model to switch between ox-alpha (1M context reasoning), Claude 3.7, and Gemini.",
			"Type /workflow ultra-workflow for deep autonomous multi-agent orchestration.",
			"Type /effort pro-max to allocate up to 64,000 thinking tokens for complex tasks.",
			"Type /mcp to connect and inspect Model Context Protocol external servers.",
		}
		selectedTip := tips[time.Now().Unix()%int64(len(tips))]
		printBoxLine(fmt.Sprintf("%s %s", BoldYellow("💡 Tip:"), GrayText(selectedTip)), cardWidth)
	} else {
		printBoxLine(GrayText("Type '/' for instant slash menu autocomplete, or enter your message."), cardWidth)
	}
	fmt.Println(GrayText("╰" + strings.Repeat("─", cardWidth-2) + "╯"))
	fmt.Println()
}
