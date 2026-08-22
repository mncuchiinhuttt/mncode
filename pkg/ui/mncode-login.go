package ui

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"mncode/pkg/agent"
	"mncode/pkg/config"

	"golang.org/x/term"
)

type signInPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signInResponse struct {
	Token string `json:"token"`
}

type generateKeyResponse struct {
	Success bool   `json:"success"`
	APIKey  string `json:"apiKey"`
}

// HandleMncodeLoginCommand logs into the mncode web account (mncode.dev, or a
// self-hosted instance) and mints a fresh CLI sync key — the "/login" command.
// Provider accounts (Antigravity, Codex...) now live under "/account login".
func HandleMncodeLoginCommand(parts []string, s *agent.Session) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Println("Usage: /login  (interactive — prompts for your mncode account email & password)")
		return
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n%s Log in to your mncode account\n", BoldPastelPink("[Login]"))
	fmt.Print("Email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)
	if email == "" {
		fmt.Println("Login cancelled.")
		return
	}

	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		fmt.Printf("%s Failed to read password: %v\n", BoldRed("[Error]"), err)
		return
	}
	password := strings.TrimSpace(string(passwordBytes))
	if password == "" {
		fmt.Println("Login cancelled.")
		return
	}

	key, err := signInAndMintKey(s.Config.GetWebBaseURL(), email, password)
	if err != nil {
		fmt.Printf("%s %v\n", BoldRed("[Login Failed]"), err)
		fmt.Println(GrayText("  No account? Register at the mncode web dashboard, then run /login again."))
		return
	}

	s.Config.TelemetryKey = key
	s.Config.SetSetting("telemetry_key", key)
	_ = config.SaveConfig(s.Config)

	fmt.Printf("%s Logged in as %s — sync key saved.\n", BoldGreen("[Success]"), Bold(email))
	fmt.Println(GrayText("  Run /status to confirm, /sync to push usage, /feedback to send feedback."))
	fmt.Println()
}

// signInAndMintKey authenticates against the web app and generates a fresh
// CLI API key for this login, using the bearer-token session it returns.
func signInAndMintKey(baseURL, email, password string) (string, error) {
	signInURL := baseURL + "/api/auth/sign-in/email"
	genKeyURL := baseURL + "/api/keys/generate"
	client := &http.Client{Timeout: 8 * time.Second}

	signInBody, _ := json.Marshal(signInPayload{Email: email, Password: password})
	req, err := http.NewRequest("POST", signInURL, bytes.NewBuffer(signInBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("connection failed: %w (make sure the web app is running at %s)", err, signInURL)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("sign-in failed: %s", string(body))
	}

	var signIn signInResponse
	if err := json.Unmarshal(body, &signIn); err != nil || signIn.Token == "" {
		return "", fmt.Errorf("unexpected sign-in response: %s", string(body))
	}

	hostname, _ := os.Hostname()
	keyName := fmt.Sprintf("CLI Login (%s)", hostname)
	genBody, _ := json.Marshal(map[string]string{"name": keyName})
	genReq, err := http.NewRequest("POST", genKeyURL, bytes.NewBuffer(genBody))
	if err != nil {
		return "", err
	}
	genReq.Header.Set("Content-Type", "application/json")
	genReq.Header.Set("Authorization", "Bearer "+signIn.Token)

	genResp, err := client.Do(genReq)
	if err != nil {
		return "", err
	}
	defer genResp.Body.Close()
	genRespBody, _ := io.ReadAll(genResp.Body)

	if genResp.StatusCode < 200 || genResp.StatusCode >= 300 {
		return "", fmt.Errorf("could not generate a sync key: %s", string(genRespBody))
	}

	var keyResp generateKeyResponse
	if err := json.Unmarshal(genRespBody, &keyResp); err != nil || keyResp.APIKey == "" {
		return "", fmt.Errorf("unexpected key response: %s", string(genRespBody))
	}

	return keyResp.APIKey, nil
}
