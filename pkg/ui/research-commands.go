package ui

import (
	"bufio"
	"context"
	"fmt"
	"mncode/pkg/agent"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"
)

// HandleResearchCommand executes the Deep Research pipeline
func HandleResearchCommand(parts []string, s *agent.Session) {
	topic := ""
	if len(parts) > 1 {
		topic = strings.TrimSpace(strings.Join(parts[1:], " "))
	} else {
		fmt.Print(BoldCyan("Enter deep research topic or question: "))
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		topic = strings.TrimSpace(line)
	}

	if topic == "" {
		fmt.Printf("\n%s Research topic cannot be empty.\n\n", BoldRed("[Error]"))
		return
	}

	showResearchBanner("[RESEARCH] Autonomous Deep Research Pipeline", topic, "Hypothesize -> Web Search -> Deep Digest -> Synthesize Report")

	startTime := time.Now()
	ctx := context.Background()
	filePath, err := s.ProcessDeepResearch(ctx, topic, false)
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("\n%s Research pipeline error: %v\n\n", BoldRed("[Error]"), err)
		return
	}

	showResearchSummaryCard("Deep Research Completed", filePath, elapsed)
}

// HandleLitRevCommand executes the Academic Literature Review pipeline
func HandleLitRevCommand(parts []string, s *agent.Session) {
	topic := ""
	if len(parts) > 1 {
		topic = strings.TrimSpace(strings.Join(parts[1:], " "))
	} else {
		fmt.Print(BoldCyan("Enter topic or paper for Academic Literature Review: "))
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		topic = strings.TrimSpace(line)
	}

	if topic == "" {
		fmt.Printf("\n%s Literature review topic cannot be empty.\n\n", BoldRed("[Error]"))
		return
	}

	showResearchBanner("[PAPERS] Academic Literature Review Pipeline", topic, "Taxonomy -> Citation Graph -> Comparison Matrix -> Open Gaps")

	startTime := time.Now()
	ctx := context.Background()
	filePath, err := s.ProcessDeepResearch(ctx, topic, true)
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("\n%s Literature review error: %v\n\n", BoldRed("[Error]"), err)
		return
	}

	showResearchSummaryCard("Literature Review Completed", filePath, elapsed)
}

func showResearchBanner(title, topic, pipeline string) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		width = 80
	}
	cardWidth := width - 2

	topBorder := fmt.Sprintf("%s %s %s",
		BoldPastelPink("╭── ["),
		Bold(title),
		BoldPastelPink("] "+strings.Repeat("─", cardWidth-visualLen(title)-10)+"╮"))

	fmt.Println()
	fmt.Println(topBorder)
	printMCPRow("", cardWidth)
	printMCPRow(fmt.Sprintf("  [STEER] %s %s", BoldCyan("Topic:   "), Bold(truncateText(topic, cardWidth-16))), cardWidth)
	printMCPRow(fmt.Sprintf("  [RESEARCH] %s %s", GrayText("Pipeline:"), GrayText(truncateText(pipeline, cardWidth-16))), cardWidth)
	printMCPRow("", cardWidth)
	fmt.Println(GrayText("╰" + strings.Repeat("─", cardWidth-2) + "╯"))
	fmt.Println()
}

func showResearchSummaryCard(title, filePath string, elapsed time.Duration) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		width = 80
	}
	cardWidth := width - 2

	linesCount := 0
	fileSizeKB := float64(0)
	if data, err := os.ReadFile(filePath); err == nil {
		linesCount = len(strings.Split(string(data), "\n"))
		fileSizeKB = float64(len(data)) / 1024.0
	}

	relPath := filePath
	if rel, err := filepath.Rel(".", filePath); err == nil {
		relPath = rel
	}

	topBorder := fmt.Sprintf("%s %s %s",
		BoldPastelPink("╭── ["),
		BoldGreen(fmt.Sprintf("[OK] %s", title)),
		BoldPastelPink("] "+strings.Repeat("─", cardWidth-visualLen(title)-12)+"╮"))

	fmt.Println()
	fmt.Println(topBorder)
	printMCPRow("", cardWidth)
	printMCPRow(fmt.Sprintf("  [FILE] %s %s (%d lines · %.1f KB)", BoldCyan("Report:  "), Bold(relPath), linesCount, fileSizeKB), cardWidth)
	printMCPRow(fmt.Sprintf("  ⏱️  %s %s", GrayText("Duration:"), GrayText(fmt.Sprintf("%.1fs", elapsed.Seconds()))), cardWidth)
	printMCPRow("", cardWidth)
	fmt.Println(GrayText("╰" + strings.Repeat("─", cardWidth-2) + "╯"))
	fmt.Println()
}
