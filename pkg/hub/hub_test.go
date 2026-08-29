package hub

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSanitizeProcessEnv(t *testing.T) {
	// Set mock host sensitive envs
	_ = os.Setenv("HOST_ANTHROPIC_API_KEY", "sk-ant-test123")
	_ = os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	defer func() {
		_ = os.Unsetenv("HOST_ANTHROPIC_API_KEY")
		_ = os.Unsetenv("DATABASE_URL")
	}()

	custom := map[string]string{
		"PORT":         "3000",
		"NODE_ENV":     "development",
		"SERVICE_NAME": "test-service",
	}

	sanitized := SanitizeProcessEnv(custom)

	for _, entry := range sanitized {
		if strings.Contains(entry, "HOST_ANTHROPIC_API_KEY") || strings.Contains(entry, "DATABASE_URL") {
			t.Fatalf("sanitized env leaked host secret: %s", entry)
		}
	}

	foundPort := false
	for _, entry := range sanitized {
		if entry == "PORT=3000" {
			foundPort = true
			break
		}
	}
	if !foundPort {
		t.Fatal("custom variable PORT=3000 not preserved in sanitized env")
	}
}

func TestTCPPortProber(t *testing.T) {
	// Start a mock listener on random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := WaitForPort(ctx, "127.0.0.1", port, 2*time.Second); err != nil {
		t.Fatalf("WaitForPort() error = %v", err)
	}
}

func TestServiceHubLifecycle(t *testing.T) {
	mgr := &Manager{
		processes: make(map[string]*SupervisedProcess),
	}

	spec := ServiceSpec{
		Name:       "test-echo-svc",
		Command:    "echo",
		Args:       []string{"hello from test"},
		TimeoutSec: 5,
	}

	info, err := mgr.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if info.PID <= 0 {
		t.Fatalf("invalid PID = %d", info.PID)
	}

	time.Sleep(100 * time.Millisecond)

	logs, err := mgr.Logs("test-echo-svc", 10, "")
	if err != nil {
		t.Fatalf("Logs() error = %v", err)
	}
	if len(logs) == 0 || !strings.Contains(logs[0], "hello from test") {
		t.Fatalf("expected logs to contain 'hello from test', got %v", logs)
	}

	psList := mgr.PS()
	if len(psList) != 1 {
		t.Fatalf("expected 1 process in PS, got %d", len(psList))
	}

	_ = mgr.Stop("test-echo-svc")
}
