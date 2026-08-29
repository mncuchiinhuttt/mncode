package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestKernelToolPersistsPythonNamespace(t *testing.T) {
	tool := &KernelTool{BaseDir: t.TempDir()}
	t.Cleanup(func() { _ = tool.Close() })
	first, err := tool.Execute(context.Background(), map[string]interface{}{
		"language": "python", "session_id": "test", "code": "value = 40\n_ = value + 2\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first, "result: 42") {
		t.Fatalf("first response = %q", first)
	}
	second, err := tool.Execute(context.Background(), map[string]interface{}{
		"language": "python", "session_id": "test", "code": "_ = value * 2\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(second, "result: 80") {
		t.Fatalf("namespace did not persist: %q", second)
	}
}

func TestKernelToolPersistsNodeNamespace(t *testing.T) {
	tool := &KernelTool{BaseDir: t.TempDir()}
	t.Cleanup(func() { _ = tool.Close() })
	first, err := tool.Execute(context.Background(), map[string]interface{}{
		"language": "node", "session_id": "test", "code": "value = 21; _ = value + 21;",
	})
	if err != nil {
		t.Skipf("node runtime unavailable: %v", err)
	}
	if !strings.Contains(first, "result: 42") {
		t.Fatalf("first response = %q", first)
	}
	second, err := tool.Execute(context.Background(), map[string]interface{}{
		"language": "node", "session_id": "test", "code": "_ = value * 2;",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(second, "result: 42") {
		t.Fatalf("node namespace did not persist: %q", second)
	}
}

func TestKernelToolSessionsDoNotBlockEachOther(t *testing.T) {
	tool := &KernelTool{BaseDir: t.TempDir()}
	t.Cleanup(func() { _ = tool.Close() })
	started := make(chan struct{})
	go func() {
		close(started)
		_, _ = tool.Execute(context.Background(), map[string]interface{}{
			"language": "python", "session_id": "slow", "code": "import time\ntime.sleep(1)\n_ = 1\n",
		})
	}()
	<-started
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"language": "python", "session_id": "fast", "code": "_ = 2\n",
	})
	if err != nil {
		t.Fatalf("independent session blocked: %v", err)
	}
	if !strings.Contains(result, "result: 2") {
		t.Fatalf("fast session result = %q", result)
	}
}

func TestKernelToolSeparatesSessions(t *testing.T) {
	tool := &KernelTool{BaseDir: t.TempDir()}
	t.Cleanup(func() { _ = tool.Close() })
	if _, err := tool.Execute(context.Background(), map[string]interface{}{
		"language": "python", "session_id": "one", "code": "value = 1\n",
	}); err != nil {
		t.Fatal(err)
	}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"language": "python", "session_id": "two", "code": "_ = 'value' in globals()\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "result: False") {
		t.Fatalf("sessions leaked state: %q", result)
	}
}

func TestKernelToolRejectsEmptyCode(t *testing.T) {
	tool := &KernelTool{}
	_, err := tool.Execute(context.Background(), map[string]interface{}{"language": "python"})
	if err == nil || !strings.Contains(err.Error(), "code is required") {
		t.Fatalf("expected code validation error, got %v", err)
	}
}

func TestKernelToolBoundsLargeResult(t *testing.T) {
	tool := &KernelTool{BaseDir: t.TempDir()}
	t.Cleanup(func() { _ = tool.Close() })
	for _, testCase := range []struct {
		language string
		code     string
	}{
		{language: "python", code: "_ = 'x' * 300000\n"},
		{language: "node", code: "_ = 'x'.repeat(300000);"},
	} {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"language": testCase.language, "session_id": testCase.language, "code": testCase.code,
		})
		if err != nil {
			t.Fatalf("%s kernel failed: %v", testCase.language, err)
		}
		if len(result) > 70000 {
			t.Fatalf("%s result exceeded output bound: %d", testCase.language, len(result))
		}
	}
}
