package ui

import (
	"bufio"
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"os"
	"strings"

	"golang.org/x/term"
)

func cyclePermissionMode(s *agent.Session) {
	switch s.Config.PermissionMode {
	case config.PermissionModeAsk, "":
		s.Config.PermissionMode = config.PermissionModeAuto
		s.Config.AutoApprove = true
	case config.PermissionModeAuto:
		s.Config.PermissionMode = config.PermissionModeBypass
		s.Config.AutoApprove = true
	case config.PermissionModeBypass:
		s.Config.PermissionMode = config.PermissionModePlan
		s.Config.AutoApprove = true
	case config.PermissionModePlan:
		s.Config.PermissionMode = config.PermissionModeAsk
		s.Config.AutoApprove = false
	}
	_ = config.SaveConfig(s.Config)
}

func readPipedLine() (string, bool) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", false
	}
	return strings.TrimRight(line, "\r\n"), true
}

// readLine reads one line terminated by '\n', bare '\r', or '\r\n'.
// bufio.Reader.ReadString('\n') alone hangs forever if the terminal is stuck
// in a state where Enter sends a raw '\r' with no '\n' (e.g. a TTY left in
// raw mode by an earlier interactive menu that didn't restore cleanly) —
// every keystroke just echoes back as ^M. Use this for any plain-text
// (non-password) prompt reached right after an interactive submenu.
func readLine(reader *bufio.Reader) string {
	var sb strings.Builder
	for {
		b, err := reader.ReadByte()
		if err != nil {
			break
		}
		if b == '\n' {
			break
		}
		if b == '\r' {
			if next, peekErr := reader.Peek(1); peekErr == nil && len(next) > 0 && next[0] == '\n' {
				_, _ = reader.ReadByte()
			}
			break
		}
		sb.WriteByte(b)
	}
	return strings.TrimSpace(sb.String())
}

func openSubagentMonitor(s *agent.Session, oldState *term.State) *term.State {
	fmt.Print("\033[1A\r\033[J")
	_ = term.Restore(int(os.Stdin.Fd()), oldState)
	OpenUltraWorkflowMonitorView(s)
	newState, _ := term.MakeRaw(int(os.Stdin.Fd()))
	return newState
}
