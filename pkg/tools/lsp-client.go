package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const maxLSPFrame = 4 << 20

type lspServer struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Reader
	incoming  chan lspEnvelope
	readErr   chan error
	writeMu   sync.Mutex
	nextID    int64
	closeOnce sync.Once
}

type lspEnvelope struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *lspRPCError    `json:"error,omitempty"`
}

type lspRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type lspLocation struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

type lspLocationLink struct {
	TargetURI            string   `json:"targetUri"`
	TargetSelectionRange lspRange `json:"targetSelectionRange"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

func startLSP(ctx context.Context, launch []string, dir string) (*lspServer, error) {
	binary, err := resolveLanguageServerBinary(launch[0], dir)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, binary, launch[1:]...)
	cmd.Dir = dir
	setProcessGroup(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}
	server := &lspServer{
		cmd: cmd, stdin: stdin, stdout: bufio.NewReaderSize(stdout, 64<<10),
		incoming: make(chan lspEnvelope, 32), readErr: make(chan error, 1),
	}
	go server.readLoop()
	return server, nil
}
func resolveLanguageServerBinary(name, dir string) (string, error) {
	if binary, err := exec.LookPath(name); err == nil {
		return binary, nil
	}
	if dir != "" {
		local := filepath.Join(dir, "node_modules", ".bin", name)
		if info, err := os.Stat(local); err == nil && !info.IsDir() {
			return local, nil
		}
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		candidate := filepath.Join(home, "go", "bin", name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s not found on PATH, workspace node_modules/.bin, or ~/go/bin; install it to use lsp_tool", name)
}

func (s *lspServer) close() {
	s.closeOnce.Do(func() {
		_ = s.stdin.Close()
		killProcessGroup(s.cmd)
		_ = s.cmd.Wait()
	})
}

func (s *lspServer) readLoop() {
	for {
		body, err := readLSPFrame(s.stdout)
		if err != nil {
			s.readErr <- err
			return
		}
		var message lspEnvelope
		if err := json.Unmarshal(body, &message); err != nil {
			continue
		}
		select {
		case s.incoming <- message:
		default:
			// A full notification queue must not block responses forever.
			if message.ID != nil {
				s.readErr <- fmt.Errorf("language server notification queue is full")
				return
			}
		}
	}
}

func readLSPFrame(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			continue
		}
		length, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || length < 0 || length > maxLSPFrame {
			return nil, fmt.Errorf("invalid LSP Content-Length")
		}
		contentLength = length
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("LSP frame has no Content-Length")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return body, nil
}
