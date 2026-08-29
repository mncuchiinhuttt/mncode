package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/provider"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

var (
	fileRegex = regexp.MustCompile(`([a-zA-Z0-9_\-\./]+\.(go|md|json|yml|yaml|sh|ps1|ts|tsx|js|py|rs|c|cpp|h))`)
)

// HandleRecapCommand generates a concise, structured session recap
func HandleRecapCommand(parts []string, s *agent.Session) {
	fmt.Printf("\n%s Synthesizing session activity & accomplishments...\n\n", BoldCyan("[Recap]"))
	PrintSessionRecapCard(s, false)
}

// CheckAndPrintPeriodicRecap is called after user turns to show periodic milestone recap
func CheckAndPrintPeriodicRecap(s *agent.Session, userTurnIndex int) {
	if userTurnIndex > 0 && userTurnIndex%6 == 0 {
		fmt.Printf("\n%s\n\n", BoldPastelPink("[INFO] Milestone reached! Here is a quick session checkpoint:"))
		PrintSessionRecapCard(s, true)
	}
}

// PrintSessionRecapCard renders a framed recap summary box
func PrintSessionRecapCard(s *agent.Session, isMilestone bool) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		width = 80
	}
	cardWidth := width - 2

	userMsgs := extractUserTopics(s.History)
	filesTouched := extractTouchedFiles(s.History)

	title := fmt.Sprintf("Session Recap · %d Turns", len(userMsgs))
	if isMilestone {
		title = fmt.Sprintf("Milestone Checkpoint · %d Turns", len(userMsgs))
	}

	topBorder := fmt.Sprintf("%s %s %s",
		BoldPastelPink("╭── ["),
		Bold(title),
		BoldPastelPink("] "+strings.Repeat("─", cardWidth-visualLen(title)-10)+"╮"))

	fmt.Println(topBorder)
	printRecapRow("", cardWidth)

	// Section 1: Topics / Goals
	printRecapRow(BoldCyan("[PIN] Key Topics & Requests:"), cardWidth)
	if len(userMsgs) == 0 {
		printRecapRow("  • No user messages yet in this session.", cardWidth)
	} else {
		limit := 4
		if len(userMsgs) < limit {
			limit = len(userMsgs)
		}
		for i := len(userMsgs) - limit; i < len(userMsgs); i++ {
			msg := userMsgs[i]
			if len([]rune(msg)) > 65 {
				msg = string([]rune(msg)[:62]) + "..."
			}
			printRecapRow(fmt.Sprintf("  • %s", msg), cardWidth)
		}
	}

	printRecapRow("", cardWidth)

	// Section 2: Files Touched
	printRecapRow(BoldGreen("[TOOLS] Files Referenced / Touched:"), cardWidth)
	if len(filesTouched) == 0 {
		printRecapRow(GrayText("  • None recorded in recent tool interactions"), cardWidth)
	} else {
		fileStr := strings.Join(filesTouched, ", ")
		if len([]rune(fileStr)) > cardWidth-8 {
			fileStr = string([]rune(fileStr)[:cardWidth-11]) + "..."
		}
		printRecapRow(fmt.Sprintf("  %s", fileStr), cardWidth)
	}

	printRecapRow("", cardWidth)

	// Section 3: Status / Next
	effStr := strings.ToUpper(s.Config.Effort)
	wfStr := strings.ToUpper(s.Config.Workflow)
	printRecapRow(fmt.Sprintf("%s %s · Workflow: %s · Model: %s",
		BoldMagenta("[ACTION] Status:"), Bold(effStr), BoldCyan(wfStr), GrayText(s.Config.Model)), cardWidth)

	printRecapRow(GrayText("╰"+strings.Repeat("─", cardWidth-2)+"╯"), cardWidth)
	fmt.Println()
}

func printRecapRow(content string, boxWidth int) {
	vLen := visualLen(content)
	pad := boxWidth - 4 - vLen
	if pad < 0 {
		pad = 0
	}
	if content != "" && strings.HasPrefix(content, "╰") {
		fmt.Println(content)
		return
	}
	fmt.Printf("%s %s%s %s\n",
		GrayText("│"),
		content,
		strings.Repeat(" ", pad),
		GrayText("│"))
}

func extractUserTopics(history []provider.Message) []string {
	var topics []string
	for _, m := range history {
		if m.Role == provider.RoleUser {
			clean := strings.TrimSpace(m.Content)
			if clean != "" && !strings.HasPrefix(clean, "/") && !strings.HasPrefix(clean, "[BTW]") {
				topics = append(topics, clean)
			}
		}
	}
	return topics
}

func extractTouchedFiles(history []provider.Message) []string {
	fileMap := make(map[string]bool)
	var list []string

	for _, m := range history {
		matches := fileRegex.FindAllString(m.Content, -1)
		for _, match := range matches {
			if !fileMap[match] && !strings.Contains(match, "http") && len(match) > 3 {
				fileMap[match] = true
				list = append(list, match)
			}
		}
	}

	if len(list) > 6 {
		list = list[len(list)-6:]
	}
	return list
}
