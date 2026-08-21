package ui

import (
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

// HandleMCPCommand dispatches /mcp subcommands (list, add, remove, restart)
func HandleMCPCommand(parts []string, s *agent.Session) {
	if s.MCP == nil {
		s.MCP = mcp.NewManager(s.WorkspaceDir)
	}

	if len(parts) <= 1 || parts[1] == "list" || parts[1] == "status" {
		showMCPStatus(s)
		return
	}

	switch parts[1] {
	case "add":
		if len(parts) < 4 {
			fmt.Printf("\n%s Usage: /mcp add <name> <command> [args...]\n", BoldCyan("💡"))
			fmt.Println("  Examples:")
			fmt.Println("    /mcp add github npx -y @modelcontextprotocol/server-github")
			fmt.Println("    /mcp add notion npx -y @modelcontextprotocol/server-notion")
			fmt.Println("    /mcp add postgres npx -y @modelcontextprotocol/server-postgres postgresql://localhost/mydb")
			fmt.Println()
			return
		}
		name := parts[2]
		cmd := parts[3]
		var args []string
		if len(parts) > 4 {
			args = parts[4:]
		}
		cfg := mcp.ServerConfig{
			Command: cmd,
			Args:    args,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		fmt.Printf("\n%s Connecting to MCP server '%s' (%s %s)...\n", BoldCyan("🔌 [MCP]"), name, cmd, strings.Join(args, " "))
		if err := s.MCP.AddServer(ctx, name, cfg); err != nil {
			fmt.Printf("%s Failed to add MCP server: %v\n\n", BoldRed("[Error]"), err)
			return
		}

		added := tools.RegisterMCPTools(s.Tools, s.MCP, ctx)
		fmt.Printf("%s Successfully connected! %s active MCP tools registered.\n\n", BoldGreen("✓"), Bold(fmt.Sprintf("%d", added)))

	case "remove", "rm", "delete":
		if len(parts) < 3 {
			fmt.Printf("\n%s Usage: /mcp remove <name>\n\n", BoldCyan("💡"))
			return
		}
		name := parts[2]
		if err := s.MCP.RemoveServer(name); err != nil {
			fmt.Printf("%s Failed to remove MCP server: %v\n\n", BoldRed("[Error]"), err)
			return
		}
		fmt.Printf("\n%s MCP server '%s' removed.\n\n", BoldGreen("✓"), name)

	case "restart", "reload":
		ctx := context.Background()
		s.MCP.Close()
		s.MCP.LoadConfig()
		fmt.Printf("\n%s Restarting all MCP servers...\n", BoldCyan("🔌 [MCP]"))
		s.MCP.StartAll(ctx)
		count := tools.RegisterMCPTools(s.Tools, s.MCP, ctx)
		fmt.Printf("%s MCP reloaded. %d tools active.\n\n", BoldGreen("✓"), count)

	default:
		showMCPStatus(s)
	}
}

func showMCPStatus(s *agent.Session) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statuses := s.MCP.GetStatus(ctx)

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

	if len(statuses) == 0 {
		printMCPRow(GrayText("  No MCP servers configured yet."), cardWidth)
		printMCPRow(fmt.Sprintf("  Run %s to connect GitHub, Notion, etc.", BoldCyan("/mcp add <name> <command>")), cardWidth)
	} else {
		for _, st := range statuses {
			statusBadge := BoldGreen("🟢 Connected")
			if !st.Connected {
				statusBadge = BoldRed("🔴 " + st.Error)
			}
			cmdStr := fmt.Sprintf("%s %s", st.Command, strings.Join(st.Args, " "))
			printMCPRow(fmt.Sprintf("  • %-18s %s", BoldCyan(st.Name), statusBadge), cardWidth)
			printMCPRow(fmt.Sprintf("    %s %s", GrayText("Command:"), GrayText(truncateText(cmdStr, cardWidth-20))), cardWidth)

			if len(st.Tools) > 0 {
				var toolNames []string
				for _, t := range st.Tools {
					toolNames = append(toolNames, t.Name)
				}
				toolSummary := strings.Join(toolNames, ", ")
				printMCPRow(fmt.Sprintf("    %s %s (%d tools)", GrayText("Tools:  "),
					Bold(truncateText(toolSummary, cardWidth-25)), len(st.Tools)), cardWidth)
			}
			printMCPRow("", cardWidth)
		}
	}

	printMCPRow(fmt.Sprintf("  %s %s", GrayText("Config:"), GrayText(s.MCP.ConfigPath)), cardWidth)
	printMCPRow(GrayText("╰"+strings.Repeat("─", cardWidth-2)+"╯"), cardWidth)
	fmt.Println()
}

func printMCPRow(content string, boxWidth int) {
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
