package browserctl_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"mncode/pkg/browserctl"
)

// TestSessionLiveNavigate is an integration smoke test that launches a real
// headless Chrome, navigates to a data: URL, reads the rendered text, and
// closes it. Skipped unless MNCODE_BROWSER_INTEGRATION=1 is set, since it
// requires a Chrome/Chromium binary and forks a real process.
func TestSessionLiveNavigate(t *testing.T) {
	if os.Getenv("MNCODE_BROWSER_INTEGRATION") != "1" {
		t.Skip("set MNCODE_BROWSER_INTEGRATION=1 to run live browser integration test")
	}

	tmpDir := t.TempDir()
	s := browserctl.NewSession(browserctl.Options{
		UserDataDir: tmpDir,
		Headless:    true,
	})
	t.Cleanup(func() { _ = s.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	url, title, err := s.Navigate(ctx, "data:text/html,<html><head><title>mncode-test</title></head><body><h1 id=\"greet\">hello mncode</h1><input id=\"box\" /></body></html>")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}
	if !strings.HasPrefix(url, "data:") {
		t.Fatalf("unexpected url: %s", url)
	}
	if title != "mncode-test" {
		t.Fatalf("unexpected title: %q", title)
	}

	text, err := s.GetText(ctx, "#greet")
	if err != nil {
		t.Fatalf("GetText failed: %v", err)
	}
	if strings.TrimSpace(text) != "hello mncode" {
		t.Fatalf("unexpected text: %q", text)
	}

	if err := s.Type(ctx, "#box", "hi there", true, false); err != nil {
		t.Fatalf("Type failed: %v", err)
	}
	val, err := s.Eval(ctx, `document.getElementById('box').value`)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	if val != "hi there" {
		t.Fatalf("unexpected input value: %q", val)
	}

	shot, err := s.Screenshot(ctx, false)
	if err != nil {
		t.Fatalf("Screenshot failed: %v", err)
	}
	if len(shot) == 0 {
		t.Fatalf("expected non-empty screenshot bytes")
	}

	if !s.IsRunning() {
		t.Fatalf("expected session to report running")
	}
}
