package ui

import (
	"bufio"
	"context"
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/mcp"
	"mncode/pkg/tools"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// HandleMCPCommand manages pre-configured GitHub & Notion MCP servers
func HandleMCPCommand(parts []string, s *agent.Session) {
	if s.MCP == nil {
		s.MCP = mcp.NewManager(s.WorkspaceDir)
	}

	if len(parts) <= 1 {
		OpenInteractiveMCPMenu(s)
		return
	}

	if parts[1] == "list" || parts[1] == "status" {
		showMCPStatus(s)
		return
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "github", "gh":
		token := ""
		if len(parts) > 2 {
			token = strings.TrimSpace(parts[2])
		} else {
			token = promptForToken("Enter GitHub Personal Access Token (ghp_...): ")
		}
		if token == "" {
			fmt.Printf("\n%s GitHub token cannot be empty.\n\n", BoldRed("[Error]"))
			return
		}
		cfg := mcp.ServerConfig{
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-github"},
			Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": token},
		}
		configureAndConnectMCP(s, "github", cfg)

	case "notion":
		token := ""
		if len(parts) > 2 {
			token = strings.TrimSpace(parts[2])
		} else {
			token = promptForToken("Enter Notion Integration Secret (secret_...): ")
		}
		if token == "" {
			fmt.Printf("\n%s Notion token cannot be empty.\n\n", BoldRed("[Error]"))
			return
		}
		cfg := mcp.ServerConfig{
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-notion"},
			Env:     map[string]string{"NOTION_API_KEY": token},
		}
		configureAndConnectMCP(s, "notion", cfg)

	case "remove", "rm", "delete":
		if len(parts) < 3 {
			fmt.Printf("\n%s Usage: /mcp remove <github|notion>\n\n", BoldCyan("💡"))
			return
		}
		name := strings.ToLower(parts[2])
		if err := s.MCP.RemoveServer(name); err != nil {
			fmt.Printf("%s Failed to remove '%s': %v\n\n", BoldRed("[Error]"), name, err)
			return
		}
		fmt.Printf("\n%s MCP server '%s' disconnected and removed.\n\n", BoldGreen("✓"), name)

	case "restart", "reload":
		ctx := context.Background()
		s.MCP.Close()
		s.MCP.LoadConfig()
		fmt.Printf("\n%s Restarting MCP servers...\n", BoldCyan("🔌 [MCP]"))
		s.MCP.StartAll(ctx)
		count := tools.RegisterMCPTools(s.Tools, s.MCP, ctx)
		fmt.Printf("%s MCP reloaded. %d tools active.\n\n", BoldGreen("✓"), count)

	default:
		showMCPStatus(s)
	}
}

func configureAndConnectMCP(s *agent.Session, name string, cfg mcp.ServerConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	fmt.Printf("\n%s Connecting to %s MCP server (npx @modelcontextprotocol/server-%s)...\n",
		BoldCyan("🔌 [MCP]"), Bold(name), name)

	if err := s.MCP.AddServer(ctx, name, cfg); err != nil {
		fmt.Printf("%s Connection failed: %v\n", BoldRed("[Error]"), err)
		fmt.Printf("Make sure Node.js (npx) is installed and the token is valid.\n\n")
		return
	}

	added := tools.RegisterMCPTools(s.Tools, s.MCP, ctx)
	fmt.Printf("%s %s MCP connected successfully! %s tools loaded.\n\n",
		BoldGreen("✓"), Bold(name), BoldGreen(fmt.Sprintf("%d", added)))
}

func promptForToken(promptStr string) string {
	fmt.Print(BoldCyan(promptStr))
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func showMCPStatus(s *agent.Session) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statuses := s.MCP.GetStatus(ctx)
	statusMap := make(map[string]mcp.ServerStatus)
	for _, st := range statuses {
		statusMap[st.Name] = st
	}

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		width = 80
	}
	cardWidth := width - 2

	title := "Model Context Protocol (MCP) Servers"
	topBorder := fmt.Sprintf("%s %s %s",
		BoldPastelPink("╭── ["),
		Bold(title),
		BoldPastelPink("] "+strings.Repeat("─", cardWidth-visualLen(title)-10)+"╮"))

	fmt.Println()
	fmt.Println(topBorder)
	printMCPRow("", cardWidth)

	// 1. GitHub MCP
	gh, ghConfigured := statusMap["github"]
	if ghConfigured && gh.Connected {
		printMCPRow(fmt.Sprintf("  🐙 %-14s %s (%d tools)", BoldCyan("GitHub MCP"), BoldGreen("🟢 Connected"), len(gh.Tools)), cardWidth)
		if len(gh.Tools) > 0 {
			var names []string
			for _, t := range gh.Tools {
				names = append(names, t.Name)
			}
			printMCPRow(fmt.Sprintf("     %s %s", GrayText("Tools:"), GrayText(truncateText(strings.Join(names, ", "), cardWidth-16))), cardWidth)
		}
	} else if ghConfigured {
		printMCPRow(fmt.Sprintf("  🐙 %-14s %s", BoldCyan("GitHub MCP"), BoldRed("🔴 "+gh.Error)), cardWidth)
	} else {
		printMCPRow(fmt.Sprintf("  🐙 %-14s %s · Run: %s", BoldCyan("GitHub MCP"), GrayText("⚪ Not Configured"), BoldCyan("/mcp github <token>")), cardWidth)
	}

	printMCPRow("", cardWidth)

	// 2. Notion MCP
	notion, notionConfigured := statusMap["notion"]
	if notionConfigured && notion.Connected {
		printMCPRow(fmt.Sprintf("  📝 %-14s %s (%d tools)", BoldCyan("Notion MCP"), BoldGreen("🟢 Connected"), len(notion.Tools)), cardWidth)
		if len(notion.Tools) > 0 {
			var names []string
			for _, t := range notion.Tools {
				names = append(names, t.Name)
			}
			printMCPRow(fmt.Sprintf("     %s %s", GrayText("Tools:"), GrayText(truncateText(strings.Join(names, ", "), cardWidth-16))), cardWidth)
		}
	} else if notionConfigured {
		printMCPRow(fmt.Sprintf("  📝 %-14s %s", BoldCyan("Notion MCP"), BoldRed("🔴 "+notion.Error)), cardWidth)
	} else {
		printMCPRow(fmt.Sprintf("  📝 %-14s %s · Run: %s", BoldCyan("Notion MCP"), GrayText("⚪ Not Configured"), BoldCyan("/mcp notion <token>")), cardWidth)
	}

	printMCPRow("", cardWidth)
	printMCPRow(fmt.Sprintf("  %s %s", GrayText("Config:"), GrayText(s.MCP.ConfigPath)), cardWidth)
	fmt.Println(GrayText("╰" + strings.Repeat("─", cardWidth-2) + "╯"))
	fmt.Println()
}

func printMCPRow(content string, boxWidth int) {
	vLen := visualLen(content)
	pad := boxWidth - 4 - vLen
	if pad < 0 {
		pad = 0
	}
	fmt.Printf("%s %s%s %s\n",
		GrayText("│"),
		content,
		strings.Repeat(" ", pad),
		GrayText("│"))
}
