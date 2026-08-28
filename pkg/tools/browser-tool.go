package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mncode/pkg/browserctl"
)

// BrowserTool gives the agent real control over a dedicated, isolated Chrome
// instance (navigate, click, type, read text/HTML, run JS, screenshot) via
// the Chrome DevTools Protocol. It only runs against mncode's own
// controlled-browser profile (see browserctl.DefaultUserDataDir) — never the
// user's personal default browser profile — unless the user explicitly
// imports cookies/history via the desktop Browser settings screen.
type BrowserTool struct {
	// Enabled gates whether the tool actually acts. When false, Execute
	// returns a clear error instead of silently launching a browser, so a
	// disabled "browser_control_enabled" setting is enforced at the tool
	// layer, not just hidden from the UI.
	Enabled func() bool
	// SessionFactory returns the shared browser session lazily so the
	// browser process is never launched until the tool actually runs.
	SessionFactory func() *browserctl.Session
}

func (b *BrowserTool) Name() string {
	return "control_browser"
}

func (b *BrowserTool) Description() string {
	return "Drive a real, visible Chrome browser: navigate to URLs, click and type into page elements, read visible text/HTML, list interactive elements, run JavaScript, take screenshots, and scroll. Use this to browse live websites, fill and submit forms, verify a running web app, or research pages that require real rendering/JS. Requires the user to enable browser control in Settings > Browser."
}

func (b *BrowserTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"navigate", "back", "forward", "reload",
					"click", "type", "press_key",
					"get_text", "get_html", "list_elements",
					"eval", "screenshot",
					"scroll", "scroll_into_view", "wait_for", "sleep",
					"state", "close",
				},
				"description": "The browser action to perform.",
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to navigate to. Required for action=navigate.",
			},
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector targeting an element. Required for click/type/scroll_into_view/wait_for; optional scope for get_text/get_html.",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to type. Required for action=type.",
			},
			"clear_first": map[string]interface{}{
				"type":        "boolean",
				"description": "For action=type: clear the field before typing. Defaults to true.",
			},
			"submit": map[string]interface{}{
				"type":        "boolean",
				"description": "For action=type: press Enter after typing.",
			},
			"key": map[string]interface{}{
				"type":        "string",
				"description": "For action=press_key: a named key (Enter, Tab, Escape, ArrowDown, ArrowUp, ArrowLeft, ArrowRight, Home, End, PageUp, PageDown, Backspace, Delete) or a literal character.",
			},
			"expression": map[string]interface{}{
				"type":        "string",
				"description": "JavaScript expression to evaluate. Required for action=eval.",
			},
			"dx": map[string]interface{}{
				"type":        "integer",
				"description": "For action=scroll: horizontal pixels to scroll (can be negative).",
			},
			"dy": map[string]interface{}{
				"type":        "integer",
				"description": "For action=scroll: vertical pixels to scroll (can be negative). Defaults to 600 if dx/dy both omitted.",
			},
			"full_page": map[string]interface{}{
				"type":        "boolean",
				"description": "For action=screenshot: capture the full scrollable page instead of just the viewport.",
			},
			"timeout_ms": map[string]interface{}{
				"type":        "integer",
				"description": "For action=wait_for/sleep: timeout or sleep duration in milliseconds.",
			},
			"max_elements": map[string]interface{}{
				"type":        "integer",
				"description": "For action=list_elements: max number of interactive elements to return (default 80).",
			},
		},
		"required": []string{"action"},
	}
}

func (b *BrowserTool) session() *browserctl.Session {
	if b.SessionFactory != nil {
		return b.SessionFactory()
	}
	return browserctl.Shared(browserctl.Options{})
}

func (b *BrowserTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if b.Enabled != nil && !b.Enabled() {
		return "", fmt.Errorf("browser control is disabled. Ask the user to enable it in Settings > Browser before using control_browser")
	}

	action, _ := args["action"].(string)
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		return "", fmt.Errorf("action is required")
	}

	s := b.session()
	if s == nil {
		return "", fmt.Errorf("browser session is unavailable")
	}

	switch action {
	case "navigate":
		url, _ := args["url"].(string)
		finalURL, title, err := s.Navigate(ctx, url)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Navigated to %s\nTitle: %s", finalURL, title), nil

	case "back":
		if err := s.Back(ctx); err != nil {
			return "", err
		}
		return describeState(ctx, s)

	case "forward":
		if err := s.Forward(ctx); err != nil {
			return "", err
		}
		return describeState(ctx, s)

	case "reload":
		if err := s.Reload(ctx); err != nil {
			return "", err
		}
		return describeState(ctx, s)

	case "click":
		selector, _ := args["selector"].(string)
		if err := s.Click(ctx, selector); err != nil {
			return "", err
		}
		return fmt.Sprintf("Clicked %s", selector), nil

	case "type":
		selector, _ := args["selector"].(string)
		text, _ := args["text"].(string)
		clearFirst := true
		if v, ok := args["clear_first"].(bool); ok {
			clearFirst = v
		}
		submit, _ := args["submit"].(bool)
		if err := s.Type(ctx, selector, text, clearFirst, submit); err != nil {
			return "", err
		}
		return fmt.Sprintf("Typed into %s%s", selector, ternary(submit, " and submitted", "")), nil

	case "press_key":
		key, _ := args["key"].(string)
		if err := s.PressKey(ctx, key); err != nil {
			return "", err
		}
		return fmt.Sprintf("Pressed key: %s", key), nil

	case "get_text":
		selector, _ := args["selector"].(string)
		text, err := s.GetText(ctx, selector)
		if err != nil {
			return "", err
		}
		return truncateText(text, 12000), nil

	case "get_html":
		selector, _ := args["selector"].(string)
		html, err := s.GetHTML(ctx, selector)
		if err != nil {
			return "", err
		}
		return truncateText(html, 12000), nil

	case "list_elements":
		max := intArg(args, "max_elements", 80)
		list, err := s.ListInteractiveElements(ctx, max)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(list) == "" {
			return "No interactive elements found on the page.", nil
		}
		return list, nil

	case "eval":
		expr, _ := args["expression"].(string)
		res, err := s.Eval(ctx, expr)
		if err != nil {
			return "", err
		}
		return truncateText(res, 8000), nil

	case "screenshot":
		fullPage, _ := args["full_page"].(bool)
		data, err := s.Screenshot(ctx, fullPage)
		if err != nil {
			return "", err
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		return fmt.Sprintf("Screenshot captured (%d bytes). data:image/png;base64,%s", len(data), truncateText(encoded, 400)), nil

	case "scroll":
		dx := intArg(args, "dx", 0)
		dy := intArg(args, "dy", 0)
		if dx == 0 && dy == 0 {
			dy = 600
		}
		if err := s.ScrollBy(ctx, dx, dy); err != nil {
			return "", err
		}
		return fmt.Sprintf("Scrolled by (%d, %d)", dx, dy), nil

	case "scroll_into_view":
		selector, _ := args["selector"].(string)
		if err := s.ScrollIntoView(ctx, selector); err != nil {
			return "", err
		}
		return fmt.Sprintf("Scrolled %s into view", selector), nil

	case "wait_for":
		selector, _ := args["selector"].(string)
		timeoutMs := intArg(args, "timeout_ms", 10000)
		if err := s.WaitForSelector(ctx, selector, time.Duration(timeoutMs)*time.Millisecond); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s is visible", selector), nil

	case "sleep":
		timeoutMs := intArg(args, "timeout_ms", 1000)
		if err := s.Sleep(ctx, time.Duration(timeoutMs)*time.Millisecond); err != nil {
			return "", err
		}
		return "Waited.", nil

	case "state":
		return describeState(ctx, s)

	case "close":
		if err := browserctl.CloseShared(); err != nil {
			return "", err
		}
		return "Browser closed.", nil

	default:
		return "", fmt.Errorf("unknown browser action: %s", action)
	}
}

func describeState(ctx context.Context, s *browserctl.Session) (string, error) {
	url, title, err := s.CurrentState(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("URL: %s\nTitle: %s", url, title), nil
}

func truncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n...[truncated, %d more characters]", len(s)-max)
}

func intArg(args map[string]interface{}, key string, fallback int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
