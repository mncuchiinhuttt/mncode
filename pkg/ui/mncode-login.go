package ui

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"mncode/pkg/agent"
	"mncode/pkg/config"
)

const loginCallbackTimeout = 5 * time.Minute

type loginResult struct {
	key string
	err error
}

// HandleMncodeLogoutCommand logs out of the mncode web account — clears the
// locally stored sync key. Reached via "/logout" with no arguments;
// "/logout <provider-account-id>" still removes a specific provider account
// (Antigravity, Codex...) from the pool, unchanged.
func HandleMncodeLogoutCommand(s *agent.Session) {
	if s.Config.GetTelemetryKey() == "" {
		fmt.Println("\nNot logged in to a mncode account.")
		return
	}

	s.Config.TelemetryKey = ""
	s.Config.SetSetting("telemetry_key", "")
	s.Config.SetSetting("last_telemetry_sync_date", "")
	_ = config.SaveConfig(s.Config)

	fmt.Printf("\n%s Logged out of your mncode account.\n", BoldGreen("[Logout]"))
	fmt.Println(GrayText("  Run /login anytime to link a new one."))
	fmt.Println()
}

// HandleMncodeLoginCommand logs into the mncode web account by opening the
// user's browser to a login page and waiting for it to redirect back to a
// short-lived local server with a freshly minted CLI sync key — no
// email/password ever typed into the terminal. Returns true on success.
// Provider accounts (Antigravity, Codex...) live under "/account login".
func HandleMncodeLoginCommand(parts []string, s *agent.Session) bool {
	state, err := randomHex(16)
	if err != nil {
		fmt.Printf("%s Could not start login: %v\n", BoldRed("[Login Error]"), err)
		return false
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("%s Could not start local login server: %v\n", BoldRed("[Login Error]"), err)
		return false
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	loginURL := fmt.Sprintf("%s/cli-login?callback=%s&state=%s", s.Config.GetWebBaseURL(), url.QueryEscape(callbackURL), state)

	resultCh := make(chan loginResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, callbackPage(false, "State mismatch — please try again from your terminal."))
			resultCh <- loginResult{err: fmt.Errorf("state mismatch (stale or spoofed callback)")}
			return
		}
		key := q.Get("key")
		if key == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, callbackPage(false, "No sync key was received."))
			resultCh <- loginResult{err: fmt.Errorf("no key in callback")}
			return
		}
		fmt.Fprint(w, callbackPage(true, ""))
		resultCh <- loginResult{key: key}
	})
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()

	fmt.Printf("\n%s Opening your browser to log in...\n", BoldPastelPink("[Login]"))
	fmt.Println(GrayText("  " + loginURL))
	if err := openBrowser(loginURL); err != nil {
		fmt.Println(GrayText("  Couldn't auto-open a browser — copy the URL above into one manually."))
	}
	fmt.Println(GrayText("  Waiting for you to finish in the browser (5 min timeout, Ctrl+C to cancel)..."))

	select {
	case res := <-resultCh:
		// Let the callback's HTTP response actually flush to the browser
		// before the deferred server.Close() tears the connection down.
		time.Sleep(1 * time.Second)
		if res.err != nil {
			fmt.Printf("%s %v\n", BoldRed("[Login Failed]"), res.err)
			return false
		}
		s.Config.TelemetryKey = res.key
		s.Config.SetSetting("telemetry_key", res.key)
		_ = config.SaveConfig(s.Config)
		fmt.Printf("%s Logged in — sync key saved.\n", BoldGreen("[Success]"))
		fmt.Println(GrayText("  Run /status to confirm, /sync to push usage, /feedback to send feedback."))
		fmt.Println()
		return true
	case <-time.After(loginCallbackTimeout):
		fmt.Printf("%s Timed out waiting for the browser login. Run /login to try again.\n", BoldRed("[Login Timeout]"))
		return false
	}
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(targetURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetURL)
	case "linux":
		cmd = exec.Command("xdg-open", targetURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

func callbackPage(ok bool, errMsg string) string {
	title, body := "You're logged in!", "You can close this tab and return to your terminal."
	color := "#ec4899"
	if !ok {
		title, body, color = "Login failed", errMsg, "#e11d48"
	}
	return fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><title>mncode</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #fdf7fb; color: #1c1420;
         display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
  .card { text-align: center; padding: 2.5rem; border-radius: 1rem; border: 1px solid #f5d7e8; max-width: 26rem; }
  h1 { color: %s; margin: 0 0 0.5rem; font-size: 1.25rem; }
  p { color: #746b82; margin: 0; }
</style></head>
<body><div class="card"><h1>%s</h1><p>%s</p></div></body></html>`, color, title, body)
}
