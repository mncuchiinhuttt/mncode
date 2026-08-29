package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// Client is a JSON-RPC 2.0 client talking to the official codex app-server over stdio.
type Client struct {
	mu            sync.Mutex
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	reqCounter    int64
	pendingReqs   map[int64]chan Response
	notifications chan Notification
	closed        bool
}

// StartAppServer launches the official codex app-server in an isolated environment.
func StartAppServer(ctx context.Context, runtimePath string, codexHome string) (*Client, error) {
	if runtimePath == "" {
		runtimePath = "codex"
	}
	if codexHome == "" {
		home, _ := os.UserHomeDir()
		codexHome = filepath.Join(home, ".mncode", "codex_home")
	}
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create codex home: %w", err)
	}

	cmd := exec.CommandContext(ctx, runtimePath, "app-server")
	cmd.Env = append(os.Environ(),
		"CODEX_HOME="+codexHome,
		"CODEX_STORAGE_MODE=keyring",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex app-server: %w", err)
	}

	c := &Client{
		cmd:           cmd,
		stdin:         stdin,
		stdout:        stdout,
		pendingReqs:   make(map[int64]chan Response),
		notifications: make(chan Notification, 64),
	}

	go c.readLoop()

	return c, nil
}

// Call sends a JSON-RPC request and waits for the response.
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	id := atomic.AddInt64(&c.reqCounter, 1)

	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		rawParams = b
	}

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  rawParams,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}
	reqBytes = append(reqBytes, '\n')

	respCh := make(chan Response, 1)
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("codex client is closed")
	}
	c.pendingReqs[id] = respCh
	_, writeErr := c.stdin.Write(reqBytes)
	c.mu.Unlock()

	if writeErr != nil {
		c.mu.Lock()
		delete(c.pendingReqs, id)
		c.mu.Unlock()
		return fmt.Errorf("write request failed: %w", writeErr)
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pendingReqs, id)
		c.mu.Unlock()
		return ctx.Err()
	case resp, ok := <-respCh:
		if !ok {
			return errors.New("server closed response channel")
		}
		if resp.Error != nil {
			return resp.Error
		}
		if result != nil && len(resp.Result) > 0 {
			return json.Unmarshal(resp.Result, result)
		}
		return nil
	}
}

// Notify sends a JSON-RPC notification.
func (c *Client) Notify(method string, params interface{}) error {
	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		rawParams = b
	}

	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
	}

	b, err := json.Marshal(notif)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("codex client is closed")
	}
	_, err = c.stdin.Write(b)
	return err
}

func (c *Client) readLoop() {
	scanner := bufio.NewScanner(c.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		if _, hasID := raw["id"]; hasID {
			var resp Response
			if err := json.Unmarshal(line, &resp); err == nil {
				c.mu.Lock()
				if ch, exists := c.pendingReqs[resp.ID]; exists {
					delete(c.pendingReqs, resp.ID)
					ch <- resp
					close(ch)
				}
				c.mu.Unlock()
			}
		} else if _, hasMethod := raw["method"]; hasMethod {
			var notif Notification
			if err := json.Unmarshal(line, &notif); err == nil {
				select {
				case c.notifications <- notif:
				default:
				}
			}
		}
	}

	c.mu.Lock()
	c.closed = true
	for id, ch := range c.pendingReqs {
		close(ch)
		delete(c.pendingReqs, id)
	}
	c.mu.Unlock()
}

// Close gracefully terminates the app-server.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	c.mu.Unlock()

	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}
