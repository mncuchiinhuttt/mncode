package orchestration

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// ProcessHandle supervises an OS child process bound to a Run.
type ProcessHandle struct {
	mu     sync.Mutex
	run    *Run
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// StartProcess spawns a background command attached to the given Run.
func StartProcess(run *Run, name string, args ...string) (*ProcessHandle, error) {
	if run == nil {
		return nil, fmt.Errorf("run is required")
	}

	cmd := exec.CommandContext(run.Context(), name, args...)
	if run.meta.WorkspaceDir != "" {
		cmd.Dir = run.meta.WorkspaceDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		_ = run.Fail(err)
		return nil, err
	}

	_ = run.Transition(StateRunning)
	run.EmitEvent("process_started", map[string]interface{}{
		"pid":  cmd.Process.Pid,
		"name": name,
		"args": args,
	})

	handle := &ProcessHandle{
		run:    run,
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}

	go handle.streamPipe(stdout, "stdout")
	go handle.streamPipe(stderr, "stderr")
	go handle.wait()

	return handle, nil
}

func (h *ProcessHandle) streamPipe(pipe io.Reader, label string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		text := scanner.Text()
		h.run.Log("[%s] %s", label, text)
		h.run.EmitEvent("process_output", map[string]string{
			"stream": label,
			"line":   text,
		})
	}
}

func (h *ProcessHandle) wait() {
	err := h.cmd.Wait()
	h.mu.Lock()
	defer h.mu.Unlock()

	exitCode := 0
	if h.cmd.ProcessState != nil {
		exitCode = h.cmd.ProcessState.ExitCode()
	}

	h.run.mu.Lock()
	h.run.exitCode = exitCode
	h.run.mu.Unlock()

	if err != nil {
		_ = h.run.Fail(fmt.Errorf("process exited with code %d: %w", exitCode, err))
	} else {
		_ = h.run.Complete(fmt.Sprintf("Process exited with code %d", exitCode), 0, 0)
	}

	h.run.EmitEvent("process_exited", map[string]interface{}{
		"exitCode": exitCode,
	})
}

// Kill terminates the child process.
func (h *ProcessHandle) Kill() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cmd != nil && h.cmd.Process != nil {
		return h.cmd.Process.Kill()
	}
	return nil
}
