package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	kernelMaxOutput = 64 << 10
	kernelMaxFrame  = 256 << 10
	kernelTimeout   = 30 * time.Second
)

type kernelProcess struct {
	language string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Reader
	stopOnce sync.Once
}

type kernelRequest struct {
	Code string `json:"code"`
}
type kernelResponse struct {
	OK     bool   `json:"ok"`
	Stdout string `json:"stdout,omitempty"`
	Stderr string `json:"stderr,omitempty"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func startKernel(language, dir string) (*kernelProcess, error) {
	if dir == "" {
		dir = "."
	}
	binary, args := "python3", []string{"-u", "-c", pythonKernelScript}
	if language == "node" {
		binary, args = "node", []string{"-e", nodeKernelScript}
	}
	path, err := exec.LookPath(binary)
	if err != nil && language == "python" {
		path, err = exec.LookPath("python")
	}
	if err != nil {
		return nil, fmt.Errorf("%s runtime is not installed", binary)
	}
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	setProcessGroup(cmd)
	cmd.Stderr = io.Discard
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}
	return &kernelProcess{language: language, cmd: cmd, stdin: stdin, stdout: bufio.NewReaderSize(stdout, kernelMaxOutput)}, nil
}

func (p *kernelProcess) stop() {
	p.stopOnce.Do(func() {
		_ = p.stdin.Close()
		killProcessGroup(p.cmd)
		_ = p.cmd.Wait()
	})
}

func (p *kernelProcess) execute(ctx context.Context, code string) (kernelResponse, error) {
	body, err := json.Marshal(kernelRequest{Code: code})
	if err != nil {
		return kernelResponse{}, err
	}
	if _, err := p.stdin.Write(append(body, '\n')); err != nil {
		return kernelResponse{}, fmt.Errorf("kernel write failed: %w", err)
	}
	responseCh := make(chan kernelResponse, 1)
	errorCh := make(chan error, 1)
	go func() {
		line, err := readKernelLine(p.stdout)
		if err != nil {
			errorCh <- err
			return
		}
		var response kernelResponse
		if err := json.Unmarshal(line, &response); err != nil {
			errorCh <- fmt.Errorf("kernel returned invalid response: %w", err)
			return
		}
		responseCh <- response
	}()
	timer := time.NewTimer(kernelTimeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return kernelResponse{}, ctx.Err()
	case <-timer.C:
		return kernelResponse{}, fmt.Errorf("kernel execution timed out after %s", kernelTimeout)
	case err := <-errorCh:
		return kernelResponse{}, err
	case response := <-responseCh:
		return response, nil
	}
}

func readKernelLine(reader *bufio.Reader) ([]byte, error) {
	line := make([]byte, 0, 4096)
	for {
		part, err := reader.ReadSlice('\n')
		line = append(line, part...)
		if len(line) > kernelMaxFrame {
			return nil, fmt.Errorf("kernel response exceeds %d-byte frame limit", kernelMaxFrame)
		}
		if err == nil {
			return line, nil
		}
		if err != bufio.ErrBufferFull {
			return nil, err
		}
	}
}

func formatKernelResponse(response kernelResponse) string {
	parts := make([]string, 0, 3)
	if response.Stdout != "" {
		parts = append(parts, "stdout:\n"+limitKernelOutput(response.Stdout))
	}
	if response.Stderr != "" {
		parts = append(parts, "stderr:\n"+limitKernelOutput(response.Stderr))
	}
	if response.Result != "" {
		parts = append(parts, "result: "+limitKernelOutput(response.Result))
	}
	if !response.OK && response.Error != "" {
		parts = append(parts, "error: "+limitKernelOutput(response.Error))
	}
	if len(parts) == 0 {
		return "ok"
	}
	return strings.Join(parts, "\n")
}

func limitKernelOutput(value string) string {
	if len(value) <= kernelMaxOutput {
		return value
	}
	return value[:kernelMaxOutput] + "\n[output truncated]"
}
