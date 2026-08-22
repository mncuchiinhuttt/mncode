package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/config"

	"golang.org/x/term"
)

// RunOnboardingIfNeeded shows a one-time first-run wizard: connect an LLM
// provider account (if none is configured yet) and optionally log into the
// mncode web account for cloud sync/feedback. Runs once ever — flips
// "onboarding_completed" in settings regardless of whether the user
// completes or skips each step, so it never nags on later launches.
func RunOnboardingIfNeeded(s *agent.Session) {
	if s.Config.GetSetting("onboarding_completed", "") == "true" {
		return
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}

	defer func() {
		s.Config.SetSetting("onboarding_completed", "true")
		_ = config.SaveConfig(s.Config)
	}()

	fmt.Println()
	fmt.Println(BoldPastelPink("╭──────────────────────────────────────────────────────────────╮"))
	fmt.Println(BoldPastelPink("│") + Bold("  Welcome to mncode — let's get you set up  ") + BoldPastelPink("│"))
	fmt.Println(BoldPastelPink("╰──────────────────────────────────────────────────────────────╯"))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	hasProvider := s.Config.APIKey != "" || (s.Accounts != nil && len(s.Accounts.Accounts) > 0)
	if !hasProvider {
		fmt.Println(Bold("1. Connect an AI provider"))
		fmt.Println(GrayText("   mncode needs at least one provider account (Antigravity/Gemini or Codex/OpenAI) to run."))
		fmt.Print("   Connect one now? [Y/n] ")
		ans, _ := reader.ReadString('\n')
		if !isNo(ans) {
			OpenInteractiveLoginMenu(s)
		} else {
			fmt.Println(GrayText("   Skipped — run /account login anytime."))
		}
	} else {
		fmt.Println(Bold("1. AI provider") + " " + GrayText("— already connected, skipping."))
	}

	fmt.Println()
	fmt.Println(Bold("2. Link your mncode web account") + GrayText(" (optional)"))
	fmt.Println(GrayText("   Enables /sync (daily usage tracking) and /feedback (send us feedback)."))
	fmt.Print("   Log in now? [y/N] ")
	ans, _ := reader.ReadString('\n')
	if isYes(ans) {
		HandleMncodeLoginCommand(nil, s)
	} else {
		fmt.Println(GrayText("   Skipped — run /login anytime."))
	}

	fmt.Println()
	fmt.Println(GrayText("You're all set. Type ") + BoldCyan("/help") + GrayText(" anytime to see available commands."))
	fmt.Println()
}

func isYes(input string) bool {
	v := strings.ToLower(strings.TrimSpace(input))
	return v == "y" || v == "yes"
}

func isNo(input string) bool {
	v := strings.ToLower(strings.TrimSpace(input))
	return v == "n" || v == "no"
}
