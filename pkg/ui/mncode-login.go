package ui

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
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
		if err := config.SaveConfig(s.Config); err != nil {
			fmt.Printf("%s Could not save the sync key: %v\n", BoldRed("[Login Failed]"), err)
			return false
		}
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

// callbackPage renders the local-server landing the browser lands on after the
// web bridge hands back the sync key. Styled after the mncode-web RMIT
// aesthetic (dark void, pink/cyan HUD, terminal log) so the handoff feels like
// one continuous product.
func callbackPage(ok bool, errMsg string) string {
	page := `<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>mncode — CLI Pairing</title>
<style>
  * { box-sizing: border-box; }
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center; padding:16px;
         background:#090a0f; color:#f4f4f2; overflow:hidden;
         font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
  .orb { position:fixed; border-radius:999px; filter:blur(90px); pointer-events:none; }
  .orb.pink { top:-120px; left:20%; width:520px; height:520px;
              background:radial-gradient(circle, rgba(244,114,182,.16), transparent 70%); }
  .orb.cyan { bottom:-140px; right:15%; width:480px; height:480px;
              background:radial-gradient(circle, rgba(56,189,248,.12), transparent 70%); }
  .dots { position:fixed; inset:0; pointer-events:none;
          background-image:radial-gradient(rgba(255,255,255,.10) 1px, transparent 1px);
          background-size:22px 22px; opacity:.35; }
  .card { position:relative; width:min(26rem, 100%); border:1px solid rgba(255,255,255,.14);
          background:#10121a; box-shadow:0 32px 80px rgba(0,0,0,.6); }
  .strip { display:flex; justify-content:space-between; align-items:center; gap:8px; padding:10px 18px;
           border-bottom:1px solid rgba(255,255,255,.08); background:#151823; }
  .eyebrow { display:flex; align-items:center; gap:8px; color:#f472b6; font-size:11px; font-weight:700;
             letter-spacing:.14em; text-transform:uppercase; border-left:2px solid #f472b6; padding-left:8px; }
  .beacon { width:6px; height:6px; border-radius:999px; background:#10b981;
            box-shadow:0 0 8px rgba(16,185,129,.7); animation:beacon 2s ease-in-out infinite; }
  .ver { font-size:10px; color:#71717a; letter-spacing:.1em; }
  .inner { padding:36px 32px 30px; text-align:center; }
  .icon { width:52px; height:52px; margin:0 auto 20px; display:flex; align-items:center; justify-content:center;
          font-size:20px; font-weight:700; color:#fff; }
  .icon.ok { background:#f472b6; box-shadow:0 12px 32px rgba(244,114,182,.35); }
  .icon.err { background:#fb7185; box-shadow:0 12px 32px rgba(251,113,133,.35); }
  .kicker { display:flex; justify-content:center; flex-wrap:wrap; gap:8px; font-size:11px; letter-spacing:.14em;
            text-transform:uppercase; color:#a1a1aa; margin-bottom:14px; }
  .kicker b { color:#f4f4f2; font-weight:600; }
  .kicker .pink { color:#f472b6; font-weight:700; }
  .pipe { color:rgba(161,161,170,.4); }
  h1 { margin:0; font-size:24px; font-weight:200; letter-spacing:-.02em; }
  h1 b { font-weight:700; }
  .log { margin:24px 0 0; text-align:left; border:1px solid rgba(255,255,255,.08); background:#151823;
         padding:14px 16px; font-size:12px; line-height:1.9; overflow-x:auto; }
  .ok { color:#34d399; }
  .err { color:#fb7185; }
  .cursor { display:inline-block; width:7px; height:13px; background:#f472b6; margin-left:5px;
            vertical-align:middle; animation:blink 1s steps(1) infinite; }
  .steps { display:flex; align-items:center; justify-content:center; gap:10px; margin-top:22px; }
  .step { display:flex; align-items:center; gap:6px; font-size:10px; letter-spacing:.08em;
          text-transform:uppercase; color:#f4f4f2; font-weight:700; }
  .step.dim { color:#71717a; }
  .chip { width:20px; height:20px; font-size:9px; display:flex; align-items:center; justify-content:center; }
  .chip.done { border:1px solid #f472b6; background:#f472b6; color:#fff; }
  .chip.todo { border:1px solid rgba(255,255,255,.16); color:#71717a; }
  .chip.fail { border:1px solid #fb7185; color:#fb7185; }
  .bar { width:40px; height:1px; background:#f472b6; }
  .bar.dim { background:rgba(255,255,255,.16); }
  @keyframes blink { 0%,49% { opacity:1; } 50%,100% { opacity:0; } }
  @keyframes beacon { 0%,100% { opacity:1; transform:scale(1); } 50% { opacity:.45; transform:scale(.82); } }
  @media (prefers-reduced-motion: reduce) { * { animation:none !important; } }
</style></head>
<body>
  <div class="orb pink"></div><div class="orb cyan"></div><div class="dots"></div>
  <div class="card">
    <div class="strip">
      <span class="eyebrow"><span class="beacon"></span>[ CLI Pairing Bridge ]</span>
      <span class="ver">v0.1.3.1</span>
    </div>
    <div class="inner">
      <div class="icon %ICON%">%ICON_TEXT%</div>
      <div class="kicker"><b>Sync Key</b><span class="pipe">|</span><b>Localhost</b><span class="pipe">|</span><span class="pink">E2E</span></div>
      <h1>%TITLE%</h1>
      <div class="log">%LOG%</div>
      <div class="steps">%STEPS%</div>
    </div>
  </div>
</body></html>`

	const successSteps = `<div class="step"><span class="chip done">&#10003;</span>Auth</div><div class="bar"></div>
      <div class="step"><span class="chip done">&#10003;</span>Keygen</div><div class="bar"></div>
      <div class="step"><span class="chip done">&#10003;</span>Redirect</div>`
	const failSteps = `<div class="step"><span class="chip fail">!</span>Auth</div><div class="bar dim"></div>
      <div class="step dim"><span class="chip todo">02</span>Keygen</div><div class="bar dim"></div>
      <div class="step dim"><span class="chip todo">03</span>Redirect</div>`

	page = strings.Replace(page, "%ICON%", map[bool]string{true: "ok", false: "err"}[ok], 1)
	page = strings.Replace(page, "%ICON_TEXT%", map[bool]string{true: "&gt;_", false: "!"}[ok], 1)
	page = strings.Replace(page, "%TITLE%", map[bool]string{
		true:  `You&#39;re <b>logged in!</b>`,
		false: `Login <b style="color:#fb7185">failed.</b>`,
	}[ok], 1)
	page = strings.Replace(page, "%LOG%", map[bool]string{
		true: `<div class="ok">&gt; session verified</div>
        <div class="ok">&gt; sync key issued</div>
        <div>&gt; returning to your terminal…<span class="cursor"></span></div>`,
		false: `<div class="err">! ` + html.EscapeString(errMsg) + `</div>
        <div class="cursor" style="background:#fb7185"></div>`,
	}[ok], 1)
	page = strings.Replace(page, "%STEPS%", map[bool]string{true: successSteps, false: failSteps}[ok], 1)
	return page
}
