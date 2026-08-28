// Package browserctl owns a single controllable browser instance (a real
// Chrome/Chromium/Edge/Brave process driven over the Chrome DevTools
// Protocol via chromedp) that the agent can navigate, inspect, and interact
// with as a tool. One process-wide session is shared by all callers so tool
// calls act on the same visible browser window across a whole agent turn.
package browserctl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// Options configures how the shared browser session launches.
type Options struct {
	// ExecPath overrides browser binary discovery (Chrome/Chromium/Edge/Brave).
	ExecPath string
	// UserDataDir is the Chrome profile directory used by the controlled
	// browser. Kept separate from the user's real browser profile by
	// default so agent browsing never touches their personal cookies/history
	// unless they explicitly import it (see Import in profile.go).
	UserDataDir string
	// Headless runs the browser without a visible window.
	Headless bool
	// IgnoreCertErrors disables TLS certificate verification in the browser.
	IgnoreCertErrors bool
	// WindowWidth/WindowHeight set the initial browser window size.
	WindowWidth  int
	WindowHeight int
}

// Session owns the lifecycle of one browser + one active tab. Safe for
// concurrent use; all tool calls serialize through mu so actions on the page
// (navigate, click, type) never interleave.
type Session struct {
	mu sync.Mutex

	opts       Options
	allocCtx   context.Context
	allocStop  context.CancelFunc
	browserCtx context.Context
	browserCancel context.CancelFunc

	started bool
	closed  bool

	lastURL   string
	lastTitle string
}

// NewSession creates an unstarted browser session. The browser process is
// only launched lazily on the first action (Start or any control call).
func NewSession(opts Options) *Session {
	if opts.WindowWidth <= 0 {
		opts.WindowWidth = 1360
	}
	if opts.WindowHeight <= 0 {
		opts.WindowHeight = 900
	}
	return &Session{opts: opts}
}

// DefaultUserDataDir returns the isolated Chrome profile directory mncode
// uses for agent-controlled browsing: ~/.mncode/browser-profile.
func DefaultUserDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "mncode-browser-profile")
	}
	return filepath.Join(home, ".mncode", "browser-profile")
}

// ensureStarted launches the browser process on first use. Caller must hold mu.
func (s *Session) ensureStarted(ctx context.Context) error {
	if s.closed {
		return fmt.Errorf("browser session was closed")
	}
	if s.started {
		return nil
	}

	dataDir := s.opts.UserDataDir
	if dataDir == "" {
		dataDir = DefaultUserDataDir()
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create browser profile dir: %w", err)
	}

	allocOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocOpts = append(allocOpts,
		chromedp.UserDataDir(dataDir),
		chromedp.WindowSize(s.opts.WindowWidth, s.opts.WindowHeight),
		chromedp.Flag("headless", s.opts.Headless),
		chromedp.Flag("new-window", true),
	)
	if s.opts.IgnoreCertErrors {
		allocOpts = append(allocOpts, chromedp.Flag("ignore-certificate-errors", true))
	}
	if s.opts.ExecPath != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(s.opts.ExecPath))
	}

	allocCtx, allocStop := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

	// IMPORTANT: chromedp's ExecAllocator ties the spawned Chrome process's
	// lifetime to the context passed into its *first* Run call (via
	// exec.CommandContext internally) — cancelling that context kills the
	// process immediately, even if it's just a short-lived warmup timeout.
	// So the warmup call below must run against browserCtx itself (whose
	// cancellation IS meant to kill the browser, via browserCancel/allocStop
	// on Close), never against a context we cancel right after use. We only
	// bound how long we're willing to WAIT for it via a separate watchdog
	// timer that does not touch browserCtx.
	startErr := make(chan error, 1)
	go func() {
		startErr <- chromedp.Run(browserCtx, chromedp.Navigate("about:blank"))
	}()
	select {
	case err := <-startErr:
		if err != nil {
			browserCancel()
			allocStop()
			return fmt.Errorf("failed to start controlled browser: %w", err)
		}
	case <-time.After(30 * time.Second):
		browserCancel()
		allocStop()
		return fmt.Errorf("failed to start controlled browser: timed out waiting for browser to become ready")
	}

	s.allocCtx = allocCtx
	s.allocStop = allocStop
	s.browserCtx = browserCtx
	s.browserCancel = browserCancel
	s.started = true
	return nil
}

// run executes chromedp actions against the active tab with a bounded
// timeout, starting the browser first if needed.
func (s *Session) run(ctx context.Context, timeout time.Duration, actions ...chromedp.Action) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureStarted(ctx); err != nil {
		return err
	}

	runCtx, cancel := context.WithTimeout(s.browserCtx, timeout)
	defer cancel()
	return chromedp.Run(runCtx, actions...)
}

// IsRunning reports whether the browser process is currently active.
func (s *Session) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started && !s.closed
}

// Close shuts down the browser process and releases all resources. Safe to
// call multiple times.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started || s.closed {
		s.closed = true
		return nil
	}
	s.closed = true
	if s.browserCancel != nil {
		s.browserCancel()
	}
	if s.allocStop != nil {
		s.allocStop()
	}
	s.started = false
	return nil
}
