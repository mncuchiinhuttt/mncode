package ui

import (
	"bufio"
	"fmt"
	"mncode/pkg/provider"
	"os"
	"strings"
	"sync"
)

func printCRLF(text string) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\n", "\r\n")
	fmt.Print(normalized)
}

func printlnCRLF(text string) {
	printCRLF(text + "\n")
}

type TerminalUI struct {
	mu          sync.Mutex
	spinner     *Spinner
	inThinking  bool
	autoApprove bool
	isTroll     bool
	streamBuf   string
	inCodeBlock bool
	codeLang    string
	hasStreamed bool
}

func NewTerminalUI(autoApprove bool) *TerminalUI {
	return &TerminalUI{
		spinner:     NewSpinner("Thinking..."),
		autoApprove: autoApprove,
		isTroll:     true,
	}
}

func (t *TerminalUI) SetTrollMode(enabled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.isTroll = enabled
}

func (t *TerminalUI) OnQueryStart() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inThinking = false
	t.streamBuf = ""
	t.inCodeBlock = false
	t.codeLang = ""
	t.hasStreamed = false
	t.spinner.ResetTimer()
	t.spinner.UpdateMessage(GetRandomTrollPhrase("thinking"))
	t.spinner.Start()
}

func (t *TerminalUI) OnToken(token string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.inThinking {
		fmt.Print(Reset + "\r\n")
		t.inThinking = false
	}
	t.spinner.Stop()
	t.hasStreamed = true

	t.streamBuf += token
	theme := GetCurrentTheme()

	for {
		idx := strings.Index(t.streamBuf, "\n")
		if idx == -1 {
			break
		}
		line := t.streamBuf[:idx]
		t.streamBuf = t.streamBuf[idx+1:]

		renderedLine := RenderMarkdownLine(line, &t.inCodeBlock, &t.codeLang, theme)
		printlnCRLF(renderedLine)
	}
}

func (t *TerminalUI) Flush() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.streamBuf) > 0 {
		theme := GetCurrentTheme()
		renderedLine := RenderMarkdownLine(t.streamBuf, &t.inCodeBlock, &t.codeLang, theme)
		printlnCRLF(renderedLine)
		t.streamBuf = ""
	}
	if t.inCodeBlock {
		theme := GetCurrentTheme()
		printlnCRLF(fmt.Sprintf("  %s", Colorize(theme.Muted, "╰──────────────────────────────────────────────────")))
		t.inCodeBlock = false
	}
	if t.hasStreamed {
		fmt.Print("\r\n")
		t.hasStreamed = false
	}
}

func (t *TerminalUI) OnThinking(thinking string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spinner.Stop()
	if !t.inThinking {
		printCRLF(Dim + "\n\033[35m[Thinking]\033[0m " + Italic)
		t.inThinking = true
	}
	printCRLF(thinking)
}

func (t *TerminalUI) OnToolCallStart(tc *provider.ToolCall) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.inThinking {
		fmt.Print(Reset + "\r\n")
		t.inThinking = false
	}
	if len(t.streamBuf) > 0 {
		theme := GetCurrentTheme()
		renderedLine := RenderMarkdownLine(t.streamBuf, &t.inCodeBlock, &t.codeLang, theme)
		printlnCRLF(renderedLine)
		t.streamBuf = ""
	}
	if t.inCodeBlock {
		theme := GetCurrentTheme()
		printlnCRLF(fmt.Sprintf("  %s", Colorize(theme.Muted, "╰──────────────────────────────────────────────────")))
		t.inCodeBlock = false
	}
	t.hasStreamed = false

	t.spinner.Stop()
	MaybeShowTrollPrank(t.isTroll)
	printCRLF(RenderToolCallFormatted(tc))
	t.spinner.UpdateMessage(GetRandomTrollPhrase(tc.Name))
	t.spinner.Start()
}

func (t *TerminalUI) OnToolCallResult(name string, result string, isError bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spinner.Stop()
	printCRLF(RenderToolResultFormatted(name, result, isError))
}

func (t *TerminalUI) OnSubagentStart(agentName, role, prompt string) {
	t.spinner.Stop()
	printCRLF(fmt.Sprintf("\n%s %s (%s)\n%s\n", BoldMagenta("[Subagent]"), Bold(agentName), role, GrayText(prompt)))
	t.spinner.UpdateMessage(fmt.Sprintf("%s (%s)", GetRandomTrollPhrase("subagent"), agentName))
	t.spinner.Start()
}

func (t *TerminalUI) OnSubagentComplete(agentName string, summary string) {
	t.spinner.Stop()
	printCRLF(fmt.Sprintf("%s %s %s\n", BoldGreen("[Subagent Done]"), Bold(agentName), GrayText(summary)))
}

func (t *TerminalUI) OnGoalDone(goal string, elapsedSecs float64, turns int, toolCount int) {
	t.spinner.Stop()
	mins := int(elapsedSecs) / 60
	secs := int(elapsedSecs) % 60
	timeStr := fmt.Sprintf("%dm %ds", mins, secs)
	if mins == 0 {
		timeStr = fmt.Sprintf("%.1fs", elapsedSecs)
	}

	fmt.Print("\r\n")
	printlnCRLF(BoldPastelPink("┌─────────────────────────────────────────────────────────────────────────────┐"))
	printlnCRLF(fmt.Sprintf("%s  %s %s · %d turns · %d tool calls%s%s",
		BoldPastelPink("│"),
		BoldGreen("[GOAL ACHIEVED]"),
		Bold(fmt.Sprintf("Completed in %s", timeStr)),
		turns,
		toolCount,
		strings.Repeat(" ", 12),
		BoldPastelPink("│")))
	printlnCRLF(BoldPastelPink("└─────────────────────────────────────────────────────────────────────────────┘"))
	fmt.Print("\r\n")
}

func (t *TerminalUI) OnError(err error) {
	t.spinner.Stop()
	printCRLF(fmt.Sprintf("\n%s %v\n", BoldRed("Error:"), err))
}

func (t *TerminalUI) ConfirmToolExecution(tc *provider.ToolCall) bool {
	if t.autoApprove {
		return true
	}
	t.spinner.Stop()
	printCRLF(fmt.Sprintf("\n%s Allow executing %s? [Y/n]: ", BoldYellow("[Permission]"), Bold(tc.Name)))
	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	return ans == "" || ans == "y" || ans == "yes"
}
