package hub

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

const maxLogBufferSize = 1000

// SupervisedProcess tracks a single running background child service.
type SupervisedProcess struct {
	mu        sync.RWMutex
	Spec      ServiceSpec
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	logBuffer []string
	linesChan chan string
	state     ProcessState
	startTime time.Time
}

func newSupervisedProcess(spec ServiceSpec) *SupervisedProcess {
	return &SupervisedProcess{
		Spec:      spec,
		logBuffer: make([]string, 0, maxLogBufferSize),
		linesChan: make(chan string, 128),
		state:     StateIdle,
	}
}

func (p *SupervisedProcess) start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state == StateRunning {
		return fmt.Errorf("service %q is already running", p.Spec.Name)
	}

	cmd := exec.Command(p.Spec.Command, p.Spec.Args...)
	if p.Spec.Cwd != "" {
		cmd.Dir = p.Spec.Cwd
	}
	cmd.Env = SanitizeProcessEnv(p.Spec.Env)

	setProcessGroup(cmd)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	p.stdin = stdinPipe

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdinPipe.Close()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		_ = stderrPipe.Close()
		return fmt.Errorf("start command: %w", err)
	}

	p.cmd = cmd
	p.state = StateRunning
	p.startTime = time.Now()

	// Drain stdout & stderr into logBuffer ring
	merged := io.MultiReader(stdoutPipe, stderrPipe)
	go p.streamLogs(merged)

	go func() {
		_ = cmd.Wait()
		p.mu.Lock()
		p.state = StateStopped
		close(p.linesChan)
		p.mu.Unlock()
	}()

	return nil
}

func (p *SupervisedProcess) streamLogs(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		p.mu.Lock()
		if len(p.logBuffer) >= maxLogBufferSize {
			p.logBuffer = p.logBuffer[1:]
		}
		p.logBuffer = append(p.logBuffer, line)
		p.mu.Unlock()

		select {
		case p.linesChan <- line:
		default:
		}
	}
}

// GetLogs returns buffered log lines with optional limit and regex filtering.
func (p *SupervisedProcess) GetLogs(limit int, grepPattern string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var filterRe *regexp.Regexp
	if grepPattern != "" {
		filterRe, _ = regexp.Compile(grepPattern)
	}

	var matched []string
	for _, l := range p.logBuffer {
		if filterRe == nil || filterRe.MatchString(l) {
			matched = append(matched, l)
		}
	}

	if limit > 0 && len(matched) > limit {
		matched = matched[len(matched)-limit:]
	}
	return matched
}

// SendText writes text to child stdin.
func (p *SupervisedProcess) SendText(text string, enter bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stdin == nil || p.state != StateRunning {
		return fmt.Errorf("service %q is not running", p.Spec.Name)
	}
	if enter {
		text += "\n"
	}
	_, err := io.WriteString(p.stdin, text)
	return err
}

// Stop gracefully shuts down the process tree.
func (p *SupervisedProcess) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil || p.state != StateRunning {
		return nil
	}

	killProcessGroup(p.cmd)
	p.state = StateStopped
	return nil
}
