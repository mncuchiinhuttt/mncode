//go:build windows

package commandutil

import (
	"fmt"
	"os/exec"
)

func prepareProcess(_ *exec.Cmd) {}

func killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid)).Run()
	return cmd.Process.Kill()
}
