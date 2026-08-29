package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	Name     string
	Config   ServerConfig
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Reader
	stderr   io.ReadCloser
	mu       sync.Mutex
	reqID    int64
	closed   bool
	waitDone chan struct{}
}

const maxMCPMessageBytes = 1 << 20

func sanitizedEnv(customEnv map[string]string) []string {
	// Whitelist of benign system environment variables
	allowedSystemKeys := map[string]bool{
		"PATH":               true,
		"HOME":               true,
		"USER":               true,
		"LOGNAME":            true,
		"SHELL":              true,
		"TMPDIR":             true,
		"TEMP":               true,
		"TMP":                true,
		"LANG":               true,
		"LC_ALL":             true,
		"LC_CTYPE":           true,
		"SYSTEMROOT":         true,
		"WINDIR":             true,
		"APPDATA":            true,
		"LOCALAPPDATA":       true,
		"PROGRAMFILES":       true,
		"PROGRAMFILES(X86)":  true,
		"COMMONPROGRAMFILES": true,
		"NODE_PATH":          true,
		"NVM_BIN":            true,
		"NVM_DIR":            true,
		"GOPATH":             true,
		"GOROOT":             true,
		"CARGO_HOME":         true,
		"RUSTUP_HOME":        true,
	}

	envMap := make(map[string]string)

	for _, item := range os.Environ() {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			k := parts[0]
			upperK := strings.ToUpper(k)
			// Filter out API keys, tokens, and credentials by default
			if strings.Contains(upperK, "KEY") ||
				strings.Contains(upperK, "SECRET") ||
				strings.Contains(upperK, "TOKEN") ||
				strings.Contains(upperK, "PASSWORD") ||
				strings.Contains(upperK, "AUTH") {
				continue
			}
			if allowedSystemKeys[upperK] {
				envMap[k] = parts[1]
			}
		}
	}

	// Apply explicitly declared server environment variables
	for k, v := range customEnv {
		envMap[k] = v
	}

	var result []string
	for k, v := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func NewClient(name string, cfg ServerConfig) (*Client, error) {
	return newClient(context.Background(), name, cfg)
}

func newClient(ctx context.Context, name string, cfg ServerConfig) (*Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.Command == "" {
		return nil, fmt.Errorf("empty command for MCP server '%s'", name)
	}
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Env = sanitizedEnv(cfg.Env)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdoutPipe.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdoutPipe.Close()
		_ = stderrPipe.Close()
		return nil, fmt.Errorf("failed to start server process: %w", err)
	}
	c := &Client{
		Name: name, Config: cfg, cmd: cmd, stdin: stdin,
		stdout: bufio.NewReaderSize(stdoutPipe, maxMCPMessageBytes),
		stderr: stderrPipe, waitDone: make(chan struct{}),
	}
	// Always drain stderr so a verbose server cannot deadlock on its pipe.
	go func() {
		_, _ = io.Copy(io.Discard, stderrPipe)
		_ = stderrPipe.Close()
	}()
	go func() {
		_ = cmd.Wait()
		close(c.waitDone)
	}()

	initReq := jsonRPCRequest{
		JSONRPC: "2.0", ID: c.nextID(), Method: "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]string{"name": "mncode", "version": "0.1.0"},
		},
	}
	var initResp jsonRPCResponse
	if err := c.sendRequest(ctx, initReq, &initResp, 10*time.Second); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("MCP handshake failed: %w", err)
	}
	notif := jsonRPCRequest{JSONRPC: "2.0", Method: "notifications/initialized"}
	if err := c.sendNotification(notif); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("MCP initialized notification failed: %w", err)
	}
	return c, nil
}

func (c *Client) nextID() int64 {
	return atomic.AddInt64(&c.reqID, 1)
}

func (c *Client) sendNotification(req jsonRPCRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("client closed")
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.stdin, "%s\n", string(data))
	return err
}

func (c *Client) sendRequest(ctx context.Context, req jsonRPCRequest, out interface{}, timeout time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("client closed")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.stdin, "%s\n", data); err != nil {
		return err
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		type readRes struct {
			line []byte
			err  error
		}
		ch := make(chan readRes, 1)
		go func() {
			line, readErr := c.stdout.ReadBytes('\n')
			ch <- readRes{line: line, err: readErr}
		}()
		select {
		case <-ctx.Done():
			c.terminateLocked()
			return ctx.Err()
		case <-timer.C:
			c.terminateLocked()
			return fmt.Errorf("timeout waiting for MCP server response")
		case res := <-ch:
			if res.err != nil {
				return res.err
			}
			var response jsonRPCResponse
			if err := json.Unmarshal(res.line, &response); err != nil {
				return fmt.Errorf("invalid MCP response: %w", err)
			}
			if !responseIDMatches(req.ID, response.ID) {
				// Servers may emit notifications while a request is pending.
				continue
			}
			return json.Unmarshal(res.line, out)
		}
	}
}

func responseIDMatches(want, got interface{}) bool {
	return fmt.Sprint(want) == fmt.Sprint(got) ||
		(fmt.Sprintf("%v", want) == fmt.Sprintf("%.0f", got))
}

func (c *Client) terminateLocked() {
	if c.closed {
		return
	}
	c.closed = true
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
}

func (c *Client) ListTools(ctx context.Context) ([]MCPToolInfo, error) {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	var resp jsonRPCResponse
	if err := c.sendRequest(ctx, req, &resp, 15*time.Second); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	rawResult, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Tools []struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			InputSchema map[string]interface{} `json:"inputSchema"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(rawResult, &parsed); err != nil {
		return nil, err
	}

	var tools []MCPToolInfo
	for _, t := range parsed.Tools {
		tools = append(tools, MCPToolInfo{
			ServerName:  c.Name,
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
	}

	var resp jsonRPCResponse
	if err := c.sendRequest(ctx, req, &resp, 60*time.Second); err != nil {
		return "", err
	}

	if resp.Error != nil {
		return "", fmt.Errorf("MCP tool error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	rawResult, err := json.Marshal(resp.Result)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}

	if err := json.Unmarshal(rawResult, &parsed); err != nil {
		return string(rawResult), nil
	}

	var textParts []string
	for _, item := range parsed.Content {
		if item.Type == "text" || item.Text != "" {
			textParts = append(textParts, item.Text)
		}
	}

	resText := strings.Join(textParts, "\n")
	if len(resText) > maxMCPMessageBytes {
		resText = resText[:maxMCPMessageBytes] + "\n...[output truncated]"
	}
	if parsed.IsError {
		return resText, fmt.Errorf("tool returned error: %s", resText)
	}
	return resText, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	if !c.closed {
		c.terminateLocked()
	}
	done := c.waitDone
	c.mu.Unlock()
	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timed out waiting for MCP server process cleanup")
		}
	}
	return nil
}
