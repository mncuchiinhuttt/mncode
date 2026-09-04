package ui

import (
	"fmt"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/budget"
)

func handleBudgetCommand(args string, s *agent.Session) {
	if s == nil {
		fmt.Println("\033[31m[Error] Active session required for budget tracking.\033[0m")
		return
	}

	if s.Budget == nil {
		s.Budget = budget.NewTracker(budget.BudgetSpec{})
	}

	arg := strings.TrimSpace(args)
	if arg == "" {
		renderBudgetStatus(s.Budget)
		return
	}

	if strings.EqualFold(arg, "clear") || strings.EqualFold(arg, "none") || strings.EqualFold(arg, "off") {
		s.Budget.SetBudget(budget.BudgetSpec{TokenLimit: 0})
		fmt.Println("\n\033[1;32m[OK] Token budget limit cleared (Unlimited mode).\033[0m")
		fmt.Println()
		return
	}

	// Extension syntax: +50k
	if strings.HasPrefix(arg, "+") {
		extSpec, err := budget.ParseBudget(strings.TrimPrefix(arg, "+"))
		if err != nil {
			fmt.Printf("\033[31m[Error] %v\033[0m\n", err)
			return
		}
		newLimit := s.Budget.Spec.TokenLimit + extSpec.TokenLimit
		s.Budget.SetBudget(budget.BudgetSpec{
			TokenLimit:  newLimit,
			IsHardStop:  extSpec.IsHardStop || s.Budget.Spec.IsHardStop,
			DollarLimit: s.Budget.Spec.DollarLimit,
		})
		fmt.Printf("\n\033[1;32m[OK] Extended budget ceiling by %d tokens. New limit: %d tokens.\033[0m\n\n", extSpec.TokenLimit, newLimit)
		return
	}

	spec, err := budget.ParseBudget(arg)
	if err != nil {
		fmt.Printf("\033[31m[Error] %v\033[0m\n", err)
		return
	}

	s.Budget.SetBudget(spec)
	modeStr := "Advisory Soft Limit (Warning at 80% and 100%)"
	if spec.IsHardStop {
		modeStr = "Hard Stop Ceiling (Immediate loop abort at limit)"
	}

	fmt.Printf("\n\033[1;32m[OK] Session Token Budget set to: %d tokens (%s)\033[0m\n", spec.TokenLimit, modeStr)
	if spec.DollarLimit > 0 {
		fmt.Printf("   \033[2mEquivalent estimated cost ceiling: $%.2f\033[0m\n", spec.DollarLimit)
	}
	fmt.Println()
}

func renderBudgetStatus(b *budget.Tracker) {
	fmt.Println("\n\033[1;36m=== Session Token Budget & Guardrails ===\033[0m")
	if b.Spec.TokenLimit <= 0 {
		fmt.Printf("Spent Tokens: %d tokens (Input: %d | Output: %d | Thinking: %d)\n", b.SpentTokens, b.InputTokens, b.OutputTokens, b.ThinkingTokens)
		fmt.Println("\n\033[2mSet a budget anytime with '/budget 100k' (soft) or '/budget 200k!' (hard stop).\033[0m")
		fmt.Println()
		return
	}

	modeStr := "Soft Advisory"
	if b.Spec.IsHardStop {
		modeStr = "\033[1;31mHard Stop [!]\033[0m"
	}

	rem, _ := b.Remaining()
	ratio := float64(b.SpentTokens) / float64(b.Spec.TokenLimit)
	if ratio > 1.0 {
		ratio = 1.0
	}

	barWidth := 30
	filled := int(ratio * float64(barWidth))
	bar := strings.Repeat("=", filled) + strings.Repeat("-", barWidth-filled)

	barColor := "\033[32m"
	if ratio >= 1.0 {
		barColor = "\033[31m"
	} else if ratio >= 0.8 {
		barColor = "\033[33m"
	}

	fmt.Printf("Budget Limit: %d tokens (%s)\n", b.Spec.TokenLimit, modeStr)
	fmt.Printf("Spent:        %d tokens (%.1f%%)\n", b.SpentTokens, (float64(b.SpentTokens)/float64(b.Spec.TokenLimit))*100)
	fmt.Printf("Remaining:    %d tokens\n", rem)
	fmt.Printf("Progress:     [%s%s\033[0m]\n\n", barColor, bar)
	fmt.Println("\033[2mUse '/budget +50k' to extend or '/budget clear' to remove limits.\033[0m")
	fmt.Println()
}
