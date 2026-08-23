package ui

import (
	"bufio"
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/mcp"
	"os"
	"strings"
)

func setupGitHubMCP(s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldPastelPink("╭── [ Setup GitHub MCP Server ] ───────────────────────────────────────────────╮"))
	fmt.Println("│ 📋 Step-by-Step Setup Instructions:                                          │")
	fmt.Println("│   1. Open https://github.com/settings/tokens (Personal access tokens classic)│")
	fmt.Println("│   2. Click 'Generate new token (classic)'                                    │")
	fmt.Println("│   3. Check the 'repo' scope (and 'read:org' if needed)                       │")
	fmt.Println("│   4. Copy the token (starts with ghp_...) and paste it below:                │")
	fmt.Println(BoldPastelPink("╰──────────────────────────────────────────────────────────────────────────────╯"))
	fmt.Println()

	fmt.Print(BoldCyan("Paste GitHub Token (ghp_...): "))
	reader := bufio.NewReader(os.Stdin)
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token == "" {
		fmt.Printf("\n%s Setup cancelled. No token provided.\n", BoldYellow("[Cancelled]"))
		return
	}

	cfg := mcp.ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-github"},
		Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": token},
	}
	configureAndConnectMCP(s, "github", cfg)
}

func setupNotionMCP(s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldPastelPink("╭── [ Setup Notion MCP Server ] ───────────────────────────────────────────────╮"))
	fmt.Println("│ 📋 Step-by-Step Setup Instructions:                                          │")
	fmt.Println("│   1. Open https://app.notion.com/developers/connections                     │")
	fmt.Println("│   2. Click '+ New integration', name it 'mncode', and select your workspace  │")
	fmt.Println("│   3. Copy the 'Internal Integration Secret' (starts with secret_...)         │")
	fmt.Println("│   4. In Notion, open pages/databases -> click '...' -> Connect to -> mncode │")
	fmt.Println(BoldPastelPink("╰──────────────────────────────────────────────────────────────────────────────╯"))
	fmt.Println()

	fmt.Print(BoldCyan("Paste Notion Secret (secret_...): "))
	reader := bufio.NewReader(os.Stdin)
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token == "" {
		fmt.Printf("\n%s Setup cancelled. No secret provided.\n", BoldYellow("[Cancelled]"))
		return
	}

	cfg := mcp.ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-notion"},
		Env:     map[string]string{"NOTION_API_KEY": token},
	}
	configureAndConnectMCP(s, "notion", cfg)
}

func selectAndRemoveMCP(s *agent.Session) {
	fmt.Println()
	fmt.Println("Disconnect an MCP Server:")
	fmt.Println("  1. GitHub MCP")
	fmt.Println("  2. Notion MCP")
	fmt.Print("\nEnter choice (1/2, or enter to cancel): ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	target := ""
	if choice == "1" || strings.EqualFold(choice, "github") {
		target = "github"
	} else if choice == "2" || strings.EqualFold(choice, "notion") {
		target = "notion"
	} else {
		fmt.Println("Cancelled.")
		return
	}

	if err := s.MCP.RemoveServer(target); err != nil {
		fmt.Printf("%s Failed to remove %s: %v\n", BoldRed("[Error]"), target, err)
		return
	}
	fmt.Printf("\n%s %s MCP server disconnected and removed.\n", BoldGreen("✓"), Bold(target))
}
