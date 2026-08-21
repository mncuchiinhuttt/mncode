package ui

import (
	"bufio"
	"context"
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/provider"
	"os"
	"strings"

	"golang.org/x/term"
)

// HandleBTWCommand handles ephemeral side questions without modifying main conversation history
func HandleBTWCommand(parts []string, s *agent.Session) {
	var query string
	if len(parts) > 1 {
		query = strings.TrimSpace(strings.Join(parts[1:], " "))
	} else {
		fmt.Printf("\n%s %s\n", BoldCyan("❯❯ [BTW]"), GrayText("Ask a quick side question (doesn't derail main task):"))
		query = readSimplePrompt("  > ")
	}

	if query == "" {
		fmt.Println("No question provided.")
		return
	}

	err := s.EnsureProvider()
	if err != nil {
		fmt.Printf("Provider error: %v\n", err)
		return
	}

	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width < 60 {
		width = 80
	}
	cardWidth := width - 2
	if cardWidth > 85 {
		cardWidth = 85
	}

	badge := "BTW · Side Question"
	dashes := cardWidth - 6 - len([]rune(badge))
	if dashes < 2 {
		dashes = 2
	}

	topBorder := fmt.Sprintf("%s %s %s",
		BoldPastelPink("╭──"),
		BoldPastelPink(badge),
		GrayText(strings.Repeat("─", dashes)+"╮"))

	fmt.Println()
	fmt.Println(topBorder)

	qContent := fmt.Sprintf("%s %s", Bold("Q:"), query)
	vLen := visualLen(qContent)
	pad := cardWidth - 4 - vLen
	if pad < 0 {
		pad = 0
	}
	fmt.Printf("%s %s%s %s\n",
		GrayText("│"),
		qContent,
		strings.Repeat(" ", pad),
		GrayText("│"))

	fmt.Println(GrayText("╰" + strings.Repeat("─", cardWidth-2) + "╯"))
	fmt.Println()

	var messages []provider.Message
	histLen := len(s.History)
	if histLen > 0 {
		start := histLen - 4
		if start < 0 {
			start = 0
		}
		messages = append(messages, s.History[start:]...)
	}

	messages = append(messages, provider.Message{
		Role:    provider.RoleUser,
		Content: "[Side Question]: " + query,
	})

	systemPrompt := "You are mncode, a helpful AI assistant. The user is asking a quick side question ('by the way' / btw) during their development workflow. Give a direct, concise, and helpful answer formatted with clean terminal markdown. Do not ask for permissions or plan complex tasks unless explicitly requested."

	ctx := context.Background()
	req := &provider.CompletionRequest{
		SystemPrompt:   systemPrompt,
		Messages:       messages,
		Model:          s.Config.Model,
		ThinkingBudget: 2048,
	}

	var sb strings.Builder
	_, err = s.Provider.Stream(ctx, req, func(event provider.StreamEvent) error {
		if event.Type == provider.EventToken && event.Text != "" {
			sb.WriteString(event.Text)
			fmt.Print(event.Text)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("\n%s %v\n", BoldRed("Error:"), err)
	}

	fmt.Println()
	fmt.Println()
	fmt.Println(GrayText(strings.Repeat("─", cardWidth)))
	fmt.Println(GrayText("  (Side questions do not consume main task history)"))
	fmt.Println()
}

func readSimplePrompt(prompt string) string {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		return strings.TrimSpace(line)
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		return strings.TrimSpace(line)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	fmt.Print(prompt)
	var input []rune
	buf := make([]byte, 16)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}
		b := buf[0]
		if b == 13 || b == 10 {
			fmt.Print("\r\n")
			break
		}
		if b == 3 || b == 27 {
			fmt.Print("\r\n")
			return ""
		}
		if b == 127 || b == 8 {
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Print("\b \b")
			}
			continue
		}
		if b >= 32 {
			r := rune(b)
			input = append(input, r)
			fmt.Print(string(r))
		}
	}
	return strings.TrimSpace(string(input))
}
