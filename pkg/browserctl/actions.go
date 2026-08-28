package browserctl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

const defaultActionTimeout = 30 * time.Second

// namedKeys maps common human-friendly key names to chromedp/kb key runes,
// so the agent can say "press Enter" instead of guessing escape sequences.
var namedKeys = map[string]string{
	"enter":      kb.Enter,
	"return":     kb.Enter,
	"tab":        kb.Tab,
	"escape":     kb.Escape,
	"esc":        kb.Escape,
	"backspace":  kb.Backspace,
	"delete":     kb.Delete,
	"arrowdown":  kb.ArrowDown,
	"arrowup":    kb.ArrowUp,
	"arrowleft":  kb.ArrowLeft,
	"arrowright": kb.ArrowRight,
	"down":       kb.ArrowDown,
	"up":         kb.ArrowUp,
	"left":       kb.ArrowLeft,
	"right":      kb.ArrowRight,
	"home":       kb.Home,
	"end":        kb.End,
	"pageup":     kb.PageUp,
	"pagedown":   kb.PageDown,
}

// Navigate loads a URL in the active tab and waits for load to settle.
func (s *Session) Navigate(ctx context.Context, url string) (string, string, error) {
	if strings.TrimSpace(url) == "" {
		return "", "", fmt.Errorf("url is required")
	}
	url = normalizeNavigateURL(url)
	var title string
	if err := s.run(ctx, defaultActionTimeout,
		chromedp.Navigate(url),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Title(&title),
	); err != nil {
		return "", "", fmt.Errorf("navigate to %s: %w", url, err)
	}
	s.mu.Lock()
	s.lastURL, s.lastTitle = url, title
	s.mu.Unlock()
	return url, title, nil
}

// schemedURLPrefixes are URL forms that already specify a scheme (or are a
// special browser page) and must be left untouched by Navigate's "add
// https://" convenience for bare domains/search-like input.
var schemedURLPrefixes = []string{
	"http://", "https://", "file://", "data:", "about:", "chrome:", "view-source:",
}

func normalizeNavigateURL(url string) string {
	lower := strings.ToLower(strings.TrimSpace(url))
	for _, prefix := range schemedURLPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return url
		}
	}
	return "https://" + url
}

// Back navigates to the previous page in browser history.
func (s *Session) Back(ctx context.Context) error {
	return s.run(ctx, defaultActionTimeout, chromedp.NavigateBack())
}

// Forward navigates to the next page in browser history.
func (s *Session) Forward(ctx context.Context) error {
	return s.run(ctx, defaultActionTimeout, chromedp.NavigateForward())
}

// Reload reloads the current page.
func (s *Session) Reload(ctx context.Context) error {
	return s.run(ctx, defaultActionTimeout, chromedp.Reload())
}

// Click clicks the first element matching the CSS selector.
func (s *Session) Click(ctx context.Context, selector string) error {
	if strings.TrimSpace(selector) == "" {
		return fmt.Errorf("selector is required")
	}
	return s.run(ctx, defaultActionTimeout,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
	)
}

// Type focuses the given selector, optionally clears it, and types text.
// When submit is true, an Enter key press is sent afterwards.
func (s *Session) Type(ctx context.Context, selector, text string, clearFirst, submit bool) error {
	if strings.TrimSpace(selector) == "" {
		return fmt.Errorf("selector is required")
	}
	actions := []chromedp.Action{
		chromedp.WaitVisible(selector, chromedp.ByQuery),
	}
	if clearFirst {
		actions = append(actions, chromedp.Clear(selector, chromedp.ByQuery))
	}
	actions = append(actions, chromedp.SendKeys(selector, text, chromedp.ByQuery))
	if submit {
		actions = append(actions, chromedp.SendKeys(selector, kb.Enter, chromedp.ByQuery))
	}
	return s.run(ctx, defaultActionTimeout, actions...)
}

// PressKey sends a named key (Enter, Tab, Escape, ArrowDown, ...) or a
// literal single character to whatever element currently has focus.
func (s *Session) PressKey(ctx context.Context, key string) error {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return fmt.Errorf("key is required")
	}
	if mapped, ok := namedKeys[strings.ToLower(trimmed)]; ok {
		trimmed = mapped
	}
	return s.run(ctx, defaultActionTimeout, chromedp.KeyEvent(trimmed))
}

// ScrollBy scrolls the page by the given pixel offsets via JS.
func (s *Session) ScrollBy(ctx context.Context, dx, dy int) error {
	expr := fmt.Sprintf("window.scrollBy(%d, %d)", dx, dy)
	return s.run(ctx, defaultActionTimeout, chromedp.Evaluate(expr, nil))
}

// ScrollIntoView scrolls the given selector into the viewport.
func (s *Session) ScrollIntoView(ctx context.Context, selector string) error {
	if strings.TrimSpace(selector) == "" {
		return fmt.Errorf("selector is required")
	}
	return s.run(ctx, defaultActionTimeout,
		chromedp.WaitReady(selector, chromedp.ByQuery),
		chromedp.ScrollIntoView(selector, chromedp.ByQuery),
	)
}

// GetText returns the rendered text content of the given selector, or the
// whole visible page body text when selector is empty.
func (s *Session) GetText(ctx context.Context, selector string) (string, error) {
	var text string
	if strings.TrimSpace(selector) == "" {
		if err := s.run(ctx, defaultActionTimeout,
			chromedp.Evaluate(`document.body ? document.body.innerText : ""`, &text),
		); err != nil {
			return "", err
		}
		return text, nil
	}
	if err := s.run(ctx, defaultActionTimeout,
		chromedp.WaitReady(selector, chromedp.ByQuery),
		chromedp.Text(selector, &text, chromedp.ByQuery, chromedp.NodeVisible),
	); err != nil {
		return "", err
	}
	return text, nil
}

// GetHTML returns the outer HTML of the given selector, or the whole
// document's HTML when selector is empty.
func (s *Session) GetHTML(ctx context.Context, selector string) (string, error) {
	var html string
	if strings.TrimSpace(selector) == "" {
		if err := s.run(ctx, defaultActionTimeout,
			chromedp.Evaluate(`document.documentElement.outerHTML`, &html),
		); err != nil {
			return "", err
		}
		return html, nil
	}
	if err := s.run(ctx, defaultActionTimeout,
		chromedp.WaitReady(selector, chromedp.ByQuery),
		chromedp.OuterHTML(selector, &html, chromedp.ByQuery),
	); err != nil {
		return "", err
	}
	return html, nil
}

// Eval runs arbitrary JavaScript in the page context and returns the
// JSON-encodable result as a string.
func (s *Session) Eval(ctx context.Context, expression string) (string, error) {
	if strings.TrimSpace(expression) == "" {
		return "", fmt.Errorf("expression is required")
	}
	var res string
	// Wrap so both expressions and statements work, and results always
	// come back as a string (JSON.stringify handles objects/arrays/etc).
	wrapped := fmt.Sprintf("(() => { const __r = (function(){ return (%s); })(); return typeof __r === 'string' ? __r : JSON.stringify(__r); })()", expression)
	if err := s.run(ctx, defaultActionTimeout, chromedp.Evaluate(wrapped, &res)); err != nil {
		return "", err
	}
	return res, nil
}

// WaitForSelector blocks until the given selector is visible, up to timeout.
func (s *Session) WaitForSelector(ctx context.Context, selector string, timeout time.Duration) error {
	if strings.TrimSpace(selector) == "" {
		return fmt.Errorf("selector is required")
	}
	if timeout <= 0 {
		timeout = defaultActionTimeout
	}
	return s.run(ctx, timeout, chromedp.WaitVisible(selector, chromedp.ByQuery))
}

// Sleep pauses for the given duration — useful for letting JS-heavy pages
// settle between actions. Capped to avoid an agent stalling the session.
func (s *Session) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	if d > 15*time.Second {
		d = 15 * time.Second
	}
	return s.run(ctx, d+5*time.Second, chromedp.Sleep(d))
}

// Screenshot captures the current viewport (or the full page when
// fullPage is true) and returns PNG bytes.
func (s *Session) Screenshot(ctx context.Context, fullPage bool) ([]byte, error) {
	var buf []byte
	var action chromedp.Action
	if fullPage {
		action = chromedp.FullScreenshot(&buf, 90)
	} else {
		action = chromedp.CaptureScreenshot(&buf)
	}
	if err := s.run(ctx, defaultActionTimeout, action); err != nil {
		return nil, err
	}
	return buf, nil
}

// CurrentState returns the active tab's URL and title.
func (s *Session) CurrentState(ctx context.Context) (url, title string, err error) {
	if err = s.run(ctx, defaultActionTimeout,
		chromedp.Location(&url),
		chromedp.Title(&title),
	); err != nil {
		return "", "", err
	}
	s.mu.Lock()
	s.lastURL, s.lastTitle = url, title
	s.mu.Unlock()
	return url, title, nil
}

// ListInteractiveElements returns a compact JSON-ish text summary of
// clickable/typable elements on the page (links, buttons, inputs) so the
// agent can decide what selector to act on next without dumping full HTML.
func (s *Session) ListInteractiveElements(ctx context.Context, max int) (string, error) {
	if max <= 0 || max > 200 {
		max = 80
	}
	script := fmt.Sprintf(`(() => {
		const sel = 'a[href], button, input, textarea, select, [role="button"], [onclick]';
		const nodes = Array.from(document.querySelectorAll(sel)).slice(0, %d);
		const out = nodes.map((el, i) => {
			const rect = el.getBoundingClientRect();
			const visible = rect.width > 0 && rect.height > 0;
			const tag = el.tagName.toLowerCase();
			const text = (el.innerText || el.value || el.getAttribute('aria-label') || el.getAttribute('placeholder') || '').trim().slice(0, 80);
			let selector = tag;
			if (el.id) selector = '#' + el.id;
			else if (el.getAttribute('name')) selector = tag + '[name="' + el.getAttribute('name') + '"]';
			return (i+1) + '. <' + tag + '> "' + text + '" selector=' + selector + (visible ? '' : ' (hidden)');
		});
		return out.join('\n');
	})()`, max)
	var res string
	if err := s.run(ctx, defaultActionTimeout, chromedp.Evaluate(script, &res)); err != nil {
		return "", err
	}
	return res, nil
}
