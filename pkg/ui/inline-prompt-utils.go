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

func openSubagentMonitor(s *agent.Session, oldState *term.State) *term.State {
	fmt.Print("\033[1A\r\033[J")
	_ = term.Restore(int(os.Stdin.Fd()), oldState)
	OpenUltraWorkflowMonitorView(s)
	newState, _ := term.MakeRaw(int(os.Stdin.Fd()))
	return newState
}
