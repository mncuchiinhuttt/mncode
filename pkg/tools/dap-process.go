package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func startDAP(ctx context.Context, workspace, program string) (*dapSession, error) {
	binary, err := resolveDebuggerBinary()
	if err != nil {
		return nil, err
	}
	debugProgram, tempDir, err := prepareDebugProgram(ctx, workspace, program)
	if err != nil {
		return nil, err
	}
	port, err := freeLocalPort()
	if err != nil {
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
		return nil, err
	}
	cmd := exec.CommandContext(context.Background(), binary, "dap", "--listen=127.0.0.1:"+port)
	cmd.Dir = workspace
	setProcessGroup(cmd)
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
		return nil, fmt.Errorf("start delve: %w", err)
	}
	conn, err := dialDAP(ctx, "127.0.0.1:"+port)
	if err != nil {
		killProcessGroup(cmd)
		_ = cmd.Wait()
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
		return nil, fmt.Errorf("connect to delve: %w", err)
	}
	session := &dapSession{
		cmd: cmd, conn: conn, tempDir: tempDir,
		responses: make(chan dapMessage, 32), events: make(chan dapMessage, 256),
		readErr: make(chan error, 1),
	}
	go session.readLoop()
	initCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := session.call(initCtx, "initialize", map[string]interface{}{
		"clientID": "mncode", "clientName": "mncode", "adapterID": "delve",
		"linesStartAt1": true, "columnsStartAt1": true, "pathFormat": "path",
	}, nil); err != nil {
		_ = session.close()
		return nil, fmt.Errorf("initialize debugger: %w", err)
	}
	launchSeq, err := session.send(initCtx, "launch", map[string]interface{}{
		"mode": "exec", "program": debugProgram, "cwd": workspace, "stopOnEntry": true,
	})
	if err != nil {
		_ = session.close()
		return nil, fmt.Errorf("launch debugger: %w", err)
	}
	if err := session.waitResponses(initCtx, map[int]bool{launchSeq: true}); err != nil {
		_ = session.close()
		return nil, fmt.Errorf("launch debugger: %w", err)
	}
	return session, nil
}

func prepareDebugProgram(ctx context.Context, workspace, program string) (string, string, error) {
	program, err := filepath.Abs(program)
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(program)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() && filepath.Ext(program) != ".go" {
		return program, "", nil
	}
	tempDir, err := os.MkdirTemp("", "mncode-dap-")
	if err != nil {
		return "", "", err
	}
	output := filepath.Join(tempDir, "debug-target")
	buildDir, packageArg := workspace, program
	if info.IsDir() {
		buildDir, packageArg = program, "."
	}
	build := exec.CommandContext(ctx, "go", "build", "-gcflags=all=-N -l", "-o", output, packageArg)
	build.Dir = buildDir
	setProcessGroup(build)
	var outputLog bytes.Buffer
	build.Stdout = &outputLog
	build.Stderr = &outputLog
	if err := build.Run(); err != nil {
		_ = os.RemoveAll(tempDir)
		message := outputLog.String()
		if len(message) > 2048 {
			message = message[:2048]
		}
		return "", "", fmt.Errorf("debug target build failed: %w: %s", err, message)
	}
	return output, tempDir, nil
}

func resolveDebuggerBinary() (string, error) {
	if binary, err := exec.LookPath("dlv"); err == nil {
		return binary, nil
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		candidate := filepath.Join(home, "go", "bin", "dlv")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("dlv not found on PATH or ~/go/bin; install Delve to use debugger")
}

func freeLocalPort() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	return fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port), nil
}

func dialDAP(ctx context.Context, address string) (net.Conn, error) {
	dialer := net.Dialer{}
	deadline := time.Now().Add(8 * time.Second)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err == nil {
			return conn, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
