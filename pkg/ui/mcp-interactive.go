package ui

import (
	"context"
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/mcp"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// OpenInteractiveMCPMenu opens an arrow-navigable menu for configuring MCP servers
func OpenInteractiveMCPMenu(s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		showMCPStatus(s)
		return
	}

	if s.MCP == nil {
		s.MCP = mcp.NewManager(s.WorkspaceDir)
	}

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		statuses := s.MCP.GetStatus(ctx)
		cancel()

		statusMap := make(map[string]mcp.ServerStatus)
		for _, st := range statuses {
			statusMap[st.Name] = st
		}

		ghStatus := GrayText("[[DISABLED] Not Configured]")
		if gh, ok := statusMap["github"]; ok && gh.Connected {
			ghStatus = BoldGreen(fmt.Sprintf("[[ACTIVE] Connected · %d tools]", len(gh.Tools)))
		}

		notionStatus := GrayText("[[DISABLED] Not Configured]")
		if notion, ok := statusMap["notion"]; ok && notion.Connected {
			notionStatus = BoldGreen(fmt.Sprintf("[[ACTIVE] Connected · %d tools]", len(notion.Tools)))
		}

		options := []struct {
			Title       string
			Status      string
			Description string
			Action      func()
		}{
			{
				Title:       "[GIT] GitHub MCP Server",
				Status:      ghStatus,
				Description: "Manage repos, issues, PRs, file contents & code search",
				Action:      func() { setupGitHubMCP(s) },
			},
			{
				Title:       "[DOC] Notion MCP Server",
				Status:      notionStatus,
				Description: "Read/write workspace pages, databases & knowledge blocks",
				Action:      func() { setupNotionMCP(s) },
			},
			{
				Title:       "[RELOAD] Reload & Test All MCP Servers",
				Status:      "",
				Description: "Restart server processes and reload tool catalog",
				Action: func() {
					HandleMCPCommand([]string{"/mcp", "restart"}, s)
				},
			},
			{
				Title:       "[DELETE]  Disconnect / Remove an MCP Server",
				Status:      "",
				Description: "Remove GitHub or Notion tokens from ~/.mncode/mcp.json",
				Action:      func() { selectAndRemoveMCP(s) },
			},
		}

		selected := 0
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			showMCPStatus(s)
			return
		}

		fmt.Print("\033[?25l")
		linesRendered := 0

		render := func() {
			if linesRendered > 0 {
				fmt.Printf("\033[%dA\r\033[J", linesRendered)
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("%s\r\n", BoldPastelPink("╭── [ Model Context Protocol (MCP) Hub ] ────────────────────────────────────────╮")))
			sb.WriteString(fmt.Sprintf("│ Select an MCP server to configure, test, or manage:                          │\r\n"))
			sb.WriteString(fmt.Sprintf("%s\r\n\r\n", BoldPastelPink("╰───────────────────────────────────────────────────────────────────────────────╯")))

			for i, opt := range options {
				statusStr := ""
				if opt.Status != "" {
					statusStr = " " + opt.Status
				}
				if i == selected {
					sb.WriteString(fmt.Sprintf("  %s %s%s\r\n", BoldCyan("❯"), Bold(opt.Title), statusStr))
					sb.WriteString(fmt.Sprintf("    %s\r\n\r\n", GrayText(opt.Description)))
				} else {
					sb.WriteString(fmt.Sprintf("    %s%s\r\n", opt.Title, statusStr))
					sb.WriteString(fmt.Sprintf("    %s\r\n\r\n", GrayText(opt.Description)))
				}
			}
			sb.WriteString(fmt.Sprintf("  %s\r\n", GrayText("(Use ↑/↓ arrows to navigate, Enter to select, Esc to exit)")))

			out := sb.String()
			linesRendered = strings.Count(out, "\n")
			fmt.Print(out)
		}

		render()

		shouldExit := false
		var chosenAction func()

		buf := make([]byte, 3)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				shouldExit = true
				break
			}

			if buf[0] == 3 || (buf[0] == 27 && n == 1) || buf[0] == 'q' || buf[0] == 'Q' {
				shouldExit = true
				break
			}
			if buf[0] == 13 || buf[0] == 10 {
				chosenAction = options[selected].Action
				break
			}

			if n == 3 && buf[0] == 27 && buf[1] == 91 {
				switch buf[2] {
				case 'A':
					if selected > 0 {
						selected--
						render()
					}
				case 'B':
					if selected < len(options)-1 {
						selected++
						render()
					}
				}
			}
		}

		fmt.Print("\033[?25h")
		_ = term.Restore(int(os.Stdin.Fd()), oldState)

		if linesRendered > 0 {
			fmt.Printf("\033[%dA\r\033[J", linesRendered)
		}

		if shouldExit || chosenAction == nil {
			break
		}

		chosenAction()
		fmt.Println()
		break
	}
}
