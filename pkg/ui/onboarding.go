package ui

import (
	"fmt"
	"os"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/config"

	"github.com/manifoldco/promptui"
	"golang.org/x/term"
)

// RunOnboardingIfNeeded shows a one-time first-run wizard. It never blocks on
// picking an LLM provider — if none is configured yet it silently falls back
// to the free OpenCode tier so the CLI works immediately. Logging into the
// mncode web account, however, is mandatory: the wizard keeps retrying the
// browser-based login until it succeeds (or the user Ctrl+C's out), and only
// then flips "onboarding_completed" — so it picks up right where it left off
// on the next launch if skipped via Ctrl+C.
func RunOnboardingIfNeeded(s *agent.Session) {
	if s.Config.GetSetting("onboarding_completed", "") == "true" {
		return
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}

	fmt.Println()
	fmt.Println(BoldPastelPink("Welcome to mncode — let's get you set up"))
	fmt.Println(GrayText(strings.Repeat("─", 58)))
	fmt.Println()

	hasProvider := s.Config.APIKey != "" || (s.Accounts != nil && len(s.Accounts.Accounts) > 0)
	if !hasProvider {
		s.Config.Provider = config.ProviderOpenCode
		s.Config.APIKey = "public"
		s.Config.BaseURL = "https://opencode.ai/zen/v1"
		if s.Config.Model == "" || s.Config.Model == "claude-3-7-sonnet-20250219" {
			s.Config.Model = "mimo-v2.5-free" // coding-specialist free model, confirmed working on OpenCode Zen
		}
		_ = s.EnsureProvider()
		_ = config.SaveConfig(s.Config)
		fmt.Println(Bold("Provider") + GrayText(" — no account connected, defaulting to the free tier (OpenCode)."))
		fmt.Println(GrayText("   Run /account login anytime to connect Antigravity or Codex for more capable models."))
	} else {
		fmt.Println(Bold("Provider") + GrayText(" — already connected, skipping."))
	}

	fmt.Println()
	fmt.Println(Bold("Link your mncode web account") + GrayText(" (required — enables /sync and /feedback)"))

	if s.Config.GetTelemetryKey() != "" {
		fmt.Println(GrayText("   Already linked, skipping."))
	} else {
		for !HandleMncodeLoginCommand(nil, s) {
			fmt.Println(GrayText("   Let's try that again."))
		}
	}

	// AI Personality & Persona Onboarding
	fmt.Println()
	fmt.Println(Bold("AI Personality & Persona"))
	brainrotPrompt := promptui.Select{
		Label: BoldCyan("Enable Brainrot Mode? (Gen Z / Sigma 10x Developer Persona)"),
		Items: []string{
			"No  - Standard Professional Dev (Clean & Serious)",
			"Yes - Gen Z / Max Rizz / Zero Cap fr fr 💀",
		},
		HideSelected: true,
	}

	bIdx, _, err := brainrotPrompt.Run()
	if err == nil && bIdx == 1 {
		s.Config.SetSetting("brainrot_mode", "true")
		fmt.Printf("   %s %s\n", BoldGreen("✓"), BoldPastelPink("Brainrot Mode enabled! (Zero cap fr fr)"))

		// Troll Mode Onboarding (only asked if Brainrot is enabled)
		fmt.Println()
		trollPrompt := promptui.Select{
			Label: BoldMagenta("Enable Troll Mode? (Harmless fake scare commands before tools)"),
			Items: []string{
				"No  - Regular Gen Z vibes (Safe & standard tool execution)",
				"Yes - Harmless fake scare pranks (Flashing fake rm -rf / 💀)",
			},
			HideSelected: true,
		}
		tIdx, _, tErr := trollPrompt.Run()
		if tErr == nil && tIdx == 1 {
			s.Config.SetSetting("troll_mode", "true")
			fmt.Printf("   %s %s\n", BoldGreen("✓"), BoldMagenta("Troll Mode enabled! (Harmless pranks active)"))
		} else {
			s.Config.SetSetting("troll_mode", "false")
			fmt.Printf("   %s %s\n", BoldGreen("✓"), GrayText("Troll Mode disabled."))
		}
	} else {
		s.Config.SetSetting("brainrot_mode", "false")
		s.Config.SetSetting("troll_mode", "false")
		fmt.Printf("   %s %s\n", BoldGreen("✓"), GrayText("Standard Professional Dev mode set."))
	}

	s.Config.SetSetting("onboarding_completed", "true")
	_ = config.SaveConfig(s.Config)

	fmt.Println()
	fmt.Println(GrayText("You're all set. Type ") + BoldCyan("/help") + GrayText(" anytime to see available commands."))
	fmt.Println()
}
