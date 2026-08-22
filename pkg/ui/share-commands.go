package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HandleShareCommand exports session transcript to shareable markdown / web format
func HandleShareCommand(parts []string, s *agent.Session) {
	if len(s.History) == 0 {
		fmt.Printf("\n%s No conversation history in this session to share.\n\n", BoldYellow("[Notice]"))
		return
	}

	shareDir := filepath.Join(s.WorkspaceDir, ".mncode", "shares")
	_ = os.MkdirAll(shareDir, 0755)

	shareID := fmt.Sprintf("session-%s-%d", s.ID, time.Now().Unix())
	shareFile := filepath.Join(shareDir, shareID+".md")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# mncode Session Share: %s\n\n", s.ID))
	sb.WriteString(fmt.Sprintf("- **Date:** %s\n", time.Now().Format(time.RFC1123)))
	sb.WriteString(fmt.Sprintf("- **Model:** %s (%s)\n", s.Config.Model, s.Config.Provider))
	sb.WriteString(fmt.Sprintf("- **Workspace:** `%s`\n", s.WorkspaceDir))
	sb.WriteString(fmt.Sprintf("- **Total Turns:** %d\n\n", len(s.History)))
	sb.WriteString("---\n\n")

	for i, m := range s.History {
		switch m.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("### 👤 User (Turn %d)\n\n%s\n\n", (i/2)+1, m.Content))
		case "assistant":
			sb.WriteString("### 🤖 mncode Assistant\n\n")
			if m.Thinking != "" {
				sb.WriteString(fmt.Sprintf("<details><summary><b>Thinking Process</b></summary>\n\n%s\n\n</details>\n\n", m.Thinking))
			}
			if m.Content != "" {
				sb.WriteString(m.Content + "\n\n")
			}
			if len(m.ToolCalls) > 0 {
				sb.WriteString("**Tool Calls:**\n")
				for _, tc := range m.ToolCalls {
					sb.WriteString(fmt.Sprintf("- `%s`: `%v`\n", tc.Name, tc.Arguments))
				}
				sb.WriteString("\n")
			}
		}
	}

	content := sb.String()
	if err := os.WriteFile(shareFile, []byte(content), 0644); err != nil {
		fmt.Printf("\n%s Failed saving share file: %v\n\n", BoldRed("[Error]"), err)
		return
	}

	_ = CopyToClipboard(content)

	fmt.Println()
	fmt.Printf("  %s %s\n", BoldGreen("✓"), Bold("Session Transcript Exported!"))
	fmt.Printf("  %s %s\n", BoldCyan("Local File:"), GrayText(shareFile))
	fmt.Printf("  %s %s\n", BoldYellow("Clipboard: "), Bold("Full Markdown transcript copied to OS clipboard!"))
	fmt.Println()
}
