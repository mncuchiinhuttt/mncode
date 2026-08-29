package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// HandleScanCommand executes full codebase scanning, architectural analysis & context priming
func HandleScanCommand(parts []string, s *agent.Session) {
	fmt.Println()
	fmt.Printf("%s Deep scanning codebase architecture, modules & symbols...\n", BoldCyan("[SEARCH] [Scan]"))

	startTime := time.Now()
	summary, err := agent.ScanCodebase(s.WorkspaceDir)
	if err != nil {
		fmt.Printf("%s Codebase scan failed: %v\n\n", BoldRed("[Error]"), err)
		return
	}

	// Prime into active session memory
	s.CodebaseMap = summary
	elapsed := time.Since(startTime)

	RenderScanResultCard(summary, elapsed)
}

// RenderScanResultCard displays a framed architectural report card
func RenderScanResultCard(summary *agent.CodebaseSummary, elapsed time.Duration) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		width = 80
	}
	cardWidth := width - 2

	title := "Codebase Architecture & Context Graph"
	topBorder := fmt.Sprintf("%s %s %s",
		BoldPastelPink("╭── ["),
		Bold(title),
		BoldPastelPink("] "+strings.Repeat("─", cardWidth-visualLen(title)-10)+"╮"))

	fmt.Println(topBorder)
	printScanRow("", cardWidth)

	// Summary stats
	printScanRow(fmt.Sprintf("%s %s · %s %s files (%s lines) · %s %s",
		Bold("Project:"), BoldCyan(summary.ProjectType),
		Bold("Scope:"), BoldGreen(fmt.Sprintf("%d", summary.TotalFiles)),
		BoldYellow(fmt.Sprintf("%d", summary.TotalLines)),
		Bold("Time:"), GrayText(fmt.Sprintf("%dms", elapsed.Milliseconds()))), cardWidth)

	if len(summary.Entrypoints) > 0 {
		printScanRow(fmt.Sprintf("%s %s", Bold("Entrypoints:"), BoldCyan(strings.Join(summary.Entrypoints, ", "))), cardWidth)
	}

	// Language breakdown
	var langItems []string
	for l, c := range summary.Languages {
		langItems = append(langItems, fmt.Sprintf("%s: %s files", l, Bold(fmt.Sprintf("%d", c))))
	}
	printScanRow(fmt.Sprintf("%s %s", Bold("Languages:  "), strings.Join(langItems, " · ")), cardWidth)

	printScanRow("", cardWidth)
	printScanRow(BoldMagenta("[DIR] Key Module & Package Architecture:"), cardWidth)

	for i, pkg := range summary.Packages {
		if i >= 8 {
			break
		}
		sampleFiles := strings.Join(pkg.Files, ", ")
		if len([]rune(sampleFiles)) > cardWidth-35 {
			sampleFiles = string([]rune(sampleFiles)[:cardWidth-38]) + "..."
		}
		pkgLine := fmt.Sprintf("  • %-22s %-12s %s",
			BoldCyan(pkg.Path),
			GrayText(fmt.Sprintf("%d lines (%d f)", pkg.Lines, pkg.FileCount)),
			GrayText(sampleFiles))
		printScanRow(pkgLine, cardWidth)
	}

	printScanRow("", cardWidth)
	printScanRow(fmt.Sprintf("%s %s",
		BoldGreen("[OK] Context Status:"),
		Bold("Architectural map primed into AI memory for zero-delay understanding.")), cardWidth)

	printScanRow(GrayText("╰"+strings.Repeat("─", cardWidth-2)+"╯"), cardWidth)
	fmt.Println()
}

func printScanRow(content string, boxWidth int) {
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
