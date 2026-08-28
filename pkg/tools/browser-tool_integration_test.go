package tools_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"mncode/pkg/browserctl"
	"mncode/pkg/tools"
)

// TestBrowserToolLiveEndToEnd exercises the actual BrowserTool.Execute path
// (the same one the agent's ReAct loop calls) end-to-end against a real
// headless Chrome: navigate, read text, type, screenshot, close. Skipped
// unless MNCODE_BROWSER_INTEGRATION=1.
func TestBrowserToolLiveEndToEnd(t *testing.T) {
	if os.Getenv("MNCODE_BROWSER_INTEGRATION") != "1" {
		t.Skip("set MNCODE_BROWSER_INTEGRATION=1 to run live browser integration test")
	}

	tmpDir := t.TempDir()
	session := browserctl.NewSession(browserctl.Options{UserDataDir: tmpDir, Headless: true})
	t.Cleanup(func() { _ = session.Close() })

	tool := &tools.BrowserTool{
		Enabled:        func() bool { return true },
		SessionFactory: func() *browserctl.Session { return session },
	}
	ctx := context.Background()

	out, err := tool.Execute(ctx, map[string]interface{}{
		"action": "navigate",
		"url":    `data:text/html,<html><head><title>mncode-e2e</title></head><body><h1 id="greet">hello agent</h1><input id="box"/></body></html>`,
	})
	if err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if !strings.Contains(out, "mncode-e2e") {
		t.Fatalf("unexpected navigate output: %s", out)
	}

	out, err = tool.Execute(ctx, map[string]interface{}{"action": "get_text", "selector": "#greet"})
	if err != nil {
		t.Fatalf("get_text: %v", err)
	}
	if strings.TrimSpace(out) != "hello agent" {
		t.Fatalf("unexpected get_text output: %q", out)
	}

	out, err = tool.Execute(ctx, map[string]interface{}{
		"action":   "type",
		"selector": "#box",
		"text":     "agent typed this",
	})
	if err != nil {
		t.Fatalf("type: %v", err)
	}
	if !strings.Contains(out, "#box") {
		t.Fatalf("unexpected type output: %s", out)
	}

	out, err = tool.Execute(ctx, map[string]interface{}{
		"action":     "eval",
		"expression": "document.getElementById('box').value",
	})
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if out != "agent typed this" {
		t.Fatalf("unexpected eval output: %q", out)
	}

	out, err = tool.Execute(ctx, map[string]interface{}{"action": "list_elements"})
	if err != nil {
		t.Fatalf("list_elements: %v", err)
	}
	if !strings.Contains(out, "input") {
		t.Fatalf("expected input element in list_elements output: %s", out)
	}

	out, err = tool.Execute(ctx, map[string]interface{}{"action": "screenshot"})
	if err != nil {
		t.Fatalf("screenshot: %v", err)
	}
	if !strings.Contains(out, "Screenshot captured") {
		t.Fatalf("unexpected screenshot output: %s", out)
	}

	// Disabled gate must block execution outright.
	disabledTool := &tools.BrowserTool{
		Enabled:        func() bool { return false },
		SessionFactory: func() *browserctl.Session { return session },
	}
	if _, err := disabledTool.Execute(ctx, map[string]interface{}{"action": "navigate", "url": "https://example.com"}); err == nil {
		t.Fatalf("expected error when browser control is disabled")
	}
}
