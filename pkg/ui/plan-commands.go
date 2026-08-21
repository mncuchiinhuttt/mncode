package ui

import (
	"bufio"
	"context"
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"
)

// HandlePlanCommand handles /plan, /plan mode, /plan list, and plan generation
func HandlePlanCommand(parts []string, s *agent.Session) {
	if len(parts) > 1 {
		sub := strings.ToLower(parts[1])
		switch sub {
		case "mode", "toggle":
			togglePlanMode(s)
			return
		case "list", "ls":
			listPlans(s)
			return
		}
	}

	task := ""
	if len(parts) > 1 {
		task = strings.TrimSpace(strings.Join(parts[1:], " "))
	} else {
		fmt.Print(BoldCyan("Enter feature or task to generate implementation plan (or 'mode' to toggle Plan Mode): "))
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		task = strings.TrimSpace(line)
	}

	if task == "" {
		fmt.Printf("\n%s Plan task cannot be empty.\n\n", BoldRed("[Error]"))
		return
	}

	if strings.EqualFold(task, "mode") || strings.EqualFold(task, "toggle") {
		togglePlanMode(s)
		return
	}

	showResearchBanner("📋 Autonomous Implementation Planner", task, "Scout Codebase ➔ Plan Overview ➔ Phase Breakdown ➔ Save ./plans/")

	startTime := time.Now()
	ctx := context.Background()
	planDir, err := s.ProcessPlanGeneration(ctx, task)
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("\n%s Plan generation error: %v\n\n", BoldRed("[Error]"), err)
		return
	}

	showPlanSummaryCard(planDir, elapsed)
}

func togglePlanMode(s *agent.Session) {
	if s.Config.PermissionMode == config.PermissionModePlan {
		s.Config.PermissionMode = config.PermissionModeAsk
		s.Config.AutoApprove = false
		_ = config.SaveConfig(s.Config)
		fmt.Printf("\n%s Plan Mode DISABLED. Switched back to %s permissions.\n\n",
			BoldCyan("[Plan Mode]"), BoldGreen("ASK"))
	} else {
		s.Config.PermissionMode = config.PermissionModePlan
		s.Config.AutoApprove = true
		_ = config.SaveConfig(s.Config)
		fmt.Printf("\n%s Strict Plan Mode ENABLED! 📝\n", BoldPastelPink("[Plan Mode]"))
		fmt.Println("  • Code modifications outside ./plans/ are blocked.")
		fmt.Println("  • Agent will explore and generate plans into ./plans/.")
		fmt.Println("  • Press 'Shift+Tab' or type '/plan mode' to switch back.")
		fmt.Println()
	}
}

func listPlans(s *agent.Session) {
	plansDir := filepath.Join(s.WorkspaceDir, "plans")
	entries, err := os.ReadDir(plansDir)
	if err != nil || len(entries) == 0 {
		fmt.Printf("\n%s No plans found in ./plans/\n\n", BoldYellow("[Notice]"))
		return
	}

	fmt.Printf("\n%s Plans in %s:\n", BoldPastelPink("📋"), Bold(plansDir))
	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Printf("  • %s\n", BoldCyan(entry.Name()))
		}
	}
	fmt.Println()
}

func showPlanSummaryCard(planDir string, elapsed time.Duration) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		width = 80
	}
	cardWidth := width - 2

	relPath := planDir
	if rel, err := filepath.Rel(".", planDir); err == nil {
		relPath = rel
	}

	title := "Implementation Plan Ready"
	topBorder := fmt.Sprintf("%s %s %s",
		BoldPastelPink("╭── ["),
		BoldGreen(fmt.Sprintf("✓ %s", title)),
		BoldPastelPink("] "+strings.Repeat("─", cardWidth-visualLen(title)-12)+"╮"))

	fmt.Println()
	fmt.Println(topBorder)
	printMCPRow("", cardWidth)
	printMCPRow(fmt.Sprintf("  📁 %s %s", BoldCyan("Plan Dir: "), Bold(relPath)), cardWidth)
	printMCPRow(fmt.Sprintf("  📄 %s %s", GrayText("Overview: "), GrayText(filepath.Join(relPath, "plan.md"))), cardWidth)
	printMCPRow(fmt.Sprintf("  ⏱️  %s %s", GrayText("Duration: "), GrayText(fmt.Sprintf("%.1fs", elapsed.Seconds()))), cardWidth)
	printMCPRow("", cardWidth)
	fmt.Println(GrayText("╰" + strings.Repeat("─", cardWidth-2) + "╯"))
	fmt.Println()
}
