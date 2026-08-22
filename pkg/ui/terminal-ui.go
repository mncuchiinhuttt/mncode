package ui

import (
	"bufio"
	"fmt"
	"mncode/pkg/provider"
	"os"
	"strings"
	"sync"
)

type TerminalUI struct {
	mu          sync.Mutex
	spinner     *Spinner
	inThinking  bool
	autoApprove bool
	isTroll     bool
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
	t.spinner.ResetTimer()
	t.spinner.UpdateMessage(GetRandomTrollPhrase("thinking"))
	t.spinner.Start()
}

func (t *TerminalUI) OnToken(token string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.inThinking {
		fmt.Print(Reset + "\n")
		t.inThinking = false
	}
	t.spinner.Stop()
	fmt.Print(token)
}

func (t *TerminalUI) OnThinking(thinking string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spinner.Stop()
	if !t.inThinking {
		fmt.Print(Dim + "\n\033[35m[Thinking]\033[0m " + Italic)
		t.inThinking = true
	}
	fmt.Print(thinking)
}

func (t *TerminalUI) OnToolCallStart(tc *provider.ToolCall) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.inThinking {
		fmt.Print(Reset + "\n")
		t.inThinking = false
	}
	t.spinner.Stop()
	MaybeShowTrollPrank(t.isTroll)
	fmt.Print(RenderToolCallFormatted(tc))
	t.spinner.UpdateMessage(GetRandomTrollPhrase(tc.Name))
	t.spinner.Start()
}

func (t *TerminalUI) OnToolCallResult(name string, result string, isError bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spinner.Stop()
	fmt.Print(RenderToolResultFormatted(name, result, isError))
}

func (t *TerminalUI) OnSubagentStart(agentName, role, prompt string) {
	t.spinner.Stop()
	fmt.Printf("\n%s %s (%s)\n%s\n", BoldMagenta("[Subagent]"), Bold(agentName), role, GrayText(prompt))
	t.spinner.UpdateMessage(fmt.Sprintf("%s (%s)", GetRandomTrollPhrase("subagent"), agentName))
	t.spinner.Start()
}

func (t *TerminalUI) OnSubagentComplete(agentName string, summary string) {
	t.spinner.Stop()
	fmt.Printf("%s %s completed task.\n", BoldGreen("[Subagent Done]"), agentName)
}

func (t *TerminalUI) OnGoalDone(goal string, elapsedSecs float64, turns int, toolCount int) {
	t.spinner.Stop()
	mins := int(elapsedSecs) / 60
	secs := int(elapsedSecs) % 60
	timeStr := fmt.Sprintf("%dm %ds", mins, secs)
	if mins == 0 {
		timeStr = fmt.Sprintf("%.1fs", elapsedSecs)
	}

	fmt.Println()
	fmt.Println(BoldPastelPink("┌─────────────────────────────────────────────────────────────────────────────┐"))
	fmt.Printf("%s  %s %s · %d turns · %d tool calls%s%s\n",
		BoldPastelPink("│"),
		BoldGreen("[GOAL ACHIEVED]"),
		Bold(fmt.Sprintf("Completed in %s", timeStr)),
		turns,
		toolCount,
		strings.Repeat(" ", 12),
		BoldPastelPink("│"))
	fmt.Println(BoldPastelPink("└─────────────────────────────────────────────────────────────────────────────┘"))
	fmt.Println()
}

func (t *TerminalUI) OnError(err error) {
	t.spinner.Stop()
	fmt.Printf("\n%s %v\n", BoldRed("Error:"), err)
}

func (t *TerminalUI) ConfirmToolExecution(tc *provider.ToolCall) bool {
	if t.autoApprove {
		return true
	}
	t.spinner.Stop()
	fmt.Printf("\n%s Allow executing %s? [Y/n]: ", BoldYellow("[Permission]"), Bold(tc.Name))
	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	return ans == "" || ans == "y" || ans == "yes"
}
