package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mncode/pkg/agent"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SharePayload struct {
	Title           string      `json:"title"`
	Model           string      `json:"model"`
	Provider        string      `json:"provider"`
	MarkdownContent string      `json:"markdownContent"`
	Messages        interface{} `json:"messages"`
}

type ShareResponse struct {
	Success  bool   `json:"success"`
	ShareID  string `json:"shareId"`
	ShareURL string `json:"shareUrl"`
	Error    string `json:"error"`
}

// HandleShareCommand exports session transcript and publishes it to mncode-web
func HandleShareCommand(parts []string, s *agent.Session) {
	if len(s.History) == 0 {
		fmt.Printf("\n%s No conversation history in this session to share.\n\n", BoldYellow("[Notice]"))
		return
	}

	shareDir := filepath.Join(s.WorkspaceDir, ".mncode", "shares")
	_ = os.MkdirAll(shareDir, 0755)

	sessionTitle := fmt.Sprintf("mncode Session (%s)", time.Now().Format("Jan 02, 15:04"))
	if len(parts) > 1 {
		sessionTitle = strings.Join(parts[1:], " ")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", sessionTitle))
	sb.WriteString(fmt.Sprintf("- **Date:** %s\n", time.Now().Format(time.RFC1123)))
	sb.WriteString(fmt.Sprintf("- **Model:** %s (%s)\n", s.Config.Model, s.Config.Provider))
	sb.WriteString(fmt.Sprintf("- **Workspace:** `%s`\n", s.WorkspaceDir))
	sb.WriteString(fmt.Sprintf("- **Total Messages:** %d\n\n", len(s.History)))
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
	shareID := fmt.Sprintf("session-%s-%d", s.ID, time.Now().Unix())
	shareFile := filepath.Join(shareDir, shareID+".md")
	_ = os.WriteFile(shareFile, []byte(content), 0644)

	// Publish to mncode-web
	baseURL := os.Getenv("MNCODE_WEB_URL")
	if baseURL == "" {
		baseURL = s.Config.GetSetting("web_url", "https://mncode.dev")
	}
	apiEndpoint := strings.TrimSuffix(baseURL, "/") + "/api/share"

	payload := SharePayload{
		Title:           sessionTitle,
		Model:           s.Config.Model,
		Provider:        string(s.Config.Provider),
		MarkdownContent: content,
		Messages:        s.History,
	}

	jsonBytes, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 8 * time.Second}
	req, _ := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "mncode-cli/v0.1.1")
	if key := s.Config.GetTelemetryKey(); key != "" {
		req.Header.Set("x-api-key", key)
	}

	resp, err := client.Do(req)
	var shareResp ShareResponse
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &shareResp)
	}

	fmt.Println()
	if shareResp.Success && shareResp.ShareURL != "" {
		_ = CopyToClipboard(shareResp.ShareURL)
		fmt.Printf("  %s %s\n", BoldGreen("✓"), Bold("Session Published to mncode Web!"))
		fmt.Printf("    %s %s\n", BoldCyan("Web URL:   "), BoldGreen(shareResp.ShareURL))
		fmt.Printf("    %s %s\n", GrayText("Local File:"), GrayText(shareFile))
		fmt.Printf("    %s %s\n\n", BoldYellow("Clipboard: "), Bold("Public Web Link copied to clipboard!"))
	} else {
		_ = CopyToClipboard(content)
		fmt.Printf("  %s %s\n", BoldGreen("✓"), Bold("Session Exported Locally!"))
		fmt.Printf("    %s %s\n", GrayText("Local File:"), GrayText(shareFile))
		fmt.Printf("    %s %s\n\n", BoldYellow("Clipboard: "), Bold("Full Markdown transcript copied to clipboard!"))
	}
}
