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
		Label: BoldCyan("Enable Brainrot Mode? (Gen Z / Sigma 10x Developer Persona + Harmless Troll Pranks)"),
		Items: []string{
			"No  - Standard Professional Dev (Clean & Serious)",
			"Yes - Gen Z / Max Rizz / Zero Cap fr fr [SIGMA] (Includes harmless troll pranks)",
		},
		HideSelected: true,
	}

	bIdx, _, err := brainrotPrompt.Run()
	if err == nil && bIdx == 1 {
		s.Config.SetSetting("brainrot_mode", "true")
		s.Config.SetSetting("troll_mode", "true")
		fmt.Printf("   %s %s\n", BoldGreen("[OK]"), BoldPastelPink("Brainrot & Troll Mode enabled! (Zero cap fr fr [SIGMA])"))
	} else {
		s.Config.SetSetting("brainrot_mode", "false")
		s.Config.SetSetting("troll_mode", "false")
		fmt.Printf("   %s %s\n", BoldGreen("[OK]"), GrayText("Standard Professional Dev mode set."))
	}

	// Shared Workspace Memory & Self-Evolving Reflection Onboarding
	fmt.Println()
	fmt.Println(Bold("Shared Workspace Memory & Self-Evolving Reflection"))
	fmt.Println(GrayText("   Enables agents to learn from test/build mistakes, remember repository conventions,\n   and share evolving insights across all chat sessions in the same workspace (Hermes style)."))
	memoryPrompt := promptui.Select{
		Label: BoldCyan("Enable Shared Workspace Memory & Self-Reflection?"),
		Items: []string{
			"Yes - Enable Shared Workspace Memory & Self-Reflective Learning (Recommended)",
			"No  - Ephemeral Sessions Only (Do not save or share lessons across chats)",
		},
		HideSelected: true,
	}

	mIdx, _, mErr := memoryPrompt.Run()
	if mErr == nil && mIdx == 0 {
		s.Config.SetSetting("shared_memory_enabled", "true")
		s.Config.SetSetting("hermes_reflection_enabled", "true")
		fmt.Printf("   %s %s\n", BoldGreen("[OK]"), Bold("Shared Workspace Memory & Hermes Reflection active."))
	} else {
		s.Config.SetSetting("shared_memory_enabled", "false")
		s.Config.SetSetting("hermes_reflection_enabled", "false")
		fmt.Printf("   %s %s\n", BoldGreen("[OK]"), GrayText("Shared memory disabled (Ephemeral mode)."))
	}

	s.Config.SetSetting("onboarding_completed", "true")
	_ = config.SaveConfig(s.Config)

	fmt.Println()
	fmt.Println(GrayText("You're all set. Type ") + BoldCyan("/help") + GrayText(" anytime to see available commands."))
	fmt.Println()
}
