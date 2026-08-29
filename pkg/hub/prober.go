package hub

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// WaitForPort polls a TCP host:port until it accepts connections or the timeout expires.
func WaitForPort(ctx context.Context, host string, port int, timeout time.Duration) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	deadline := time.Now().Add(timeout)
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}

		conn, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}

	return fmt.Errorf("readiness timeout (%s) waiting for TCP port %s", timeout, addr)
}

// WaitForLogRegex scans incoming process log lines until the regex pattern matches.
func WaitForLogRegex(ctx context.Context, linesChan <-chan string, pattern string, timeout time.Duration) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid readyLog regex %q: %w", pattern, err)
	}

	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("readiness timeout (%s) waiting for log regex %q", timeout, pattern)
		case line, ok := <-linesChan:
			if !ok {
				return fmt.Errorf("process stdout closed before matching readyLog regex %q", pattern)
			}
			if re.MatchString(line) {
				return nil
			}
		}
	}
}
