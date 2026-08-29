package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
	"mncode/pkg/agent"
	"mncode/pkg/config"
)

// HandleSearchCommand processes /search and /search-setup commands.
func HandleSearchCommand(parts []string, s *agent.Session) {
	if len(parts) > 0 && strings.EqualFold(parts[0], "/search-setup") {
		openSearchSetupWizard(s)
		return
	}
	if len(parts) > 1 {
		arg := strings.ToLower(parts[1])
		if arg == "setup" || arg == "config" {
			openSearchSetupWizard(s)
			return
		}
		if arg == "auto" || arg == "brave" || arg == "tavily" || arg == "antigravity" || arg == "google" || arg == "duckduckgo" || arg == "ddg" {
			if arg == "ddg" {
				arg = "duckduckgo"
			}
			if arg == "google" {
				arg = "antigravity"
			}
			s.Config.SearchEngine = arg
			s.Config.SetSetting("search_engine", arg)
			if err := config.SaveConfig(s.Config); err != nil {
				fmt.Printf("\n%s Could not save search engine: %v\n\n", BoldRed("[Error]"), err)
				return
			}
			fmt.Printf("\n%s Web Search Engine set to: %s\n\n", BoldGreen("[OK]"), BoldCyan(arg))
			return
		}
	}

	showSearchEngineStatus(s)
}

func showSearchEngineStatus(s *agent.Session) {
	activeEngine := s.Config.GetSearchEngine()
	braveKey := s.Config.GetBraveAPIKey()
	tavilyKey := s.Config.GetTavilyAPIKey()

	hasGoogle := s.Config.Provider == config.ProviderAntigravity ||
		strings.HasPrefix(s.Config.APIKey, "ya29.") ||
		(s.Accounts != nil && s.Accounts.GetActiveAccount("antigravity") != nil)

	fmt.Println()
	fmt.Println(BoldPastelPink("╭── [ Web Search Engines & Grounding ] ───────────────────────────────────────╮"))
	fmt.Printf("│  Active Engine: %-60s │\n", BoldCyan(strings.ToUpper(activeEngine)))
	fmt.Println("│                                                                              │")
	fmt.Printf("│  1. %-24s Status: %-37s │\n", Bold("Tavily AI Search"), formatKeyStatus(tavilyKey))
	fmt.Printf("│  2. %-24s Status: %-37s │\n", Bold("Brave Search API"), formatKeyStatus(braveKey))
	fmt.Printf("│  3. %-24s Status: %-37s │\n", Bold("Google Grounding"), formatGoogleStatus(hasGoogle))
	fmt.Printf("│  4. %-24s Status: %-37s │\n", Bold("DuckDuckGo (Free)"), BoldGreen("Available (Built-in No-Key)"))
	fmt.Println("│                                                                              │")
	fmt.Println("│  Commands:                                                                   │")
	fmt.Println("│    /search setup          - Interactive API key setup wizard                 │")
	fmt.Println("│    /search <engine>       - Switch engine (auto, brave, tavily, antigravity) │")
	fmt.Println(BoldPastelPink("╰──────────────────────────────────────────────────────────────────────────────╯"))
	fmt.Println()
}

func formatKeyStatus(key string) string {
	if strings.TrimSpace(key) != "" {
		return BoldGreen("Configured")
	}
	return BoldYellow("Not set (Run /search setup)")
}

func formatGoogleStatus(hasGoogle bool) string {
	if hasGoogle {
		return BoldGreen("Connected (Active OAuth)")
	}
	return BoldYellow("Connect via /login antigravity")
}

func openSearchSetupWizard(s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldCyan("Web Search Engine Setup:"))
	fmt.Println("  1. Setup Brave Search API  (Free 2,000 queries/month)")
	fmt.Println("  2. Setup Tavily Search API (Free 1,000 queries/month)")
	fmt.Println("  3. Setup Google Grounding  (Via Antigravity login)")
	fmt.Println("  4. Cancel")
	fmt.Print("\nEnter choice (1-4): ")
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		setupBraveSearchGuide(s)
	case "2":
		setupTavilySearchGuide(s)
	case "3":
		setupAntigravityGuide(s)
	default:
		fmt.Println("Setup cancelled.")
	}
}

func setupBraveSearchGuide(s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldPastelPink("╭── [ Setup Brave Search API ] ───────────────────────────────────────────────╮"))
	fmt.Println("│ [PLAN] Step-by-Step Instructions:                                                │")
	fmt.Println("│   1. Open https://brave.com/search/api/ in your browser                      │")
	fmt.Println("│   2. Sign up / Log in and get your Free API Key (2,000 queries/month free)  │")
	fmt.Println("│   3. Copy your API Key (starts with BSA...)                                  │")
	fmt.Println(BoldPastelPink("╰──────────────────────────────────────────────────────────────────────────────╯"))
	fmt.Println()

	key, err := readSearchSecret(BoldCyan("Paste Brave Search API Key: "))
	if err != nil {
		fmt.Printf("\n%s Could not read key: %v\n", BoldRed("[Error]"), err)
		return
	}
	key = strings.TrimSpace(key)

	if key == "" {
		fmt.Printf("\n%s Cancelled. No key entered.\n", BoldYellow("[Cancelled]"))
		return
	}

	s.Config.BraveAPIKey = key
	delete(s.Config.Settings, "brave_api_key")
	if err := config.SaveConfig(s.Config); err != nil {
		fmt.Printf("\n%s Could not save Brave Search API key: %v\n", BoldRed("[Error]"), err)
		return
	}
	fmt.Printf("\n%s Brave Search API key saved locally.\n", BoldGreen("[OK]"))
}

func setupTavilySearchGuide(s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldPastelPink("╭── [ Setup Tavily AI Search ] ───────────────────────────────────────────────╮"))
	fmt.Println("│ [PLAN] Step-by-Step Instructions:                                                │")
	fmt.Println("│   1. Open https://tavily.com in your browser                                 │")
	fmt.Println("│   2. Sign up / Log in and get your API Key (1,000 queries/month free)        │")
	fmt.Println("│   3. Copy your API Key (starts with tvly-...)                                │")
	fmt.Println(BoldPastelPink("╰──────────────────────────────────────────────────────────────────────────────╯"))
	fmt.Println()

	key, err := readSearchSecret(BoldCyan("Paste Tavily API Key: "))
	if err != nil {
		fmt.Printf("\n%s Could not read key: %v\n", BoldRed("[Error]"), err)
		return
	}
	key = strings.TrimSpace(key)

	if key == "" {
		fmt.Printf("\n%s Cancelled. No key entered.\n", BoldYellow("[Cancelled]"))
		return
	}

	s.Config.TavilyAPIKey = key
	delete(s.Config.Settings, "tavily_api_key")
	if err := config.SaveConfig(s.Config); err != nil {
		fmt.Printf("\n%s Could not save Tavily API key: %v\n", BoldRed("[Error]"), err)
		return
	}
	fmt.Printf("\n%s Tavily Search API key saved locally.\n", BoldGreen("[OK]"))
}

func setupAntigravityGuide(s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldPastelPink("╭── [ Setup Google Search Grounding ] ────────────────────────────────────────╮"))
	fmt.Println("│ [PLAN] Instructions:                                                             │")
	fmt.Println("│   Run `/login antigravity` or `/account import` to authenticate Google.      │")
	fmt.Println("│   Google Grounding uses native Google search results with Gemini models.     │")
	fmt.Println(BoldPastelPink("╰──────────────────────────────────────────────────────────────────────────────╯"))
	fmt.Println()
}

func readSearchSecret(prompt string) (string, error) {
	fmt.Print(prompt)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		value, err := term.ReadPassword(fd)
		fmt.Println()
		return string(value), err
	}
	reader := bufio.NewReader(os.Stdin)
	return reader.ReadString('\n')
}
