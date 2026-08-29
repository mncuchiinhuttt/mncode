package accounts

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"
)

const (
	// Official Antigravity OAuth desktop client credentials
	defaultAntigravityClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	defaultAntigravityClientSecret = "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"
)

type GoogleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// StartAntigravityWebLogin initiates a browser-based Google OAuth flow
func StartAntigravityWebLogin(store *Store) (*Account, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		clientID = defaultAntigravityClientID
	}
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = defaultAntigravityClientSecret
	}

	// 1. Generate PKCE Code Verifier & Challenge
	verifierBytes := make([]byte, 32)
	_, _ = rand.Read(verifierBytes)
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	h := sha256.New()
	h.Write([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	// 2. Bind local listener on 127.0.0.1 (try 8085 then dynamic port)
	listener, err := net.Listen("tcp", "127.0.0.1:8085")
	if err != nil {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, fmt.Errorf("failed to start local callback server: %w", err)
		}
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d", port)

	scopes := "openid https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/cclog https://www.googleapis.com/auth/experimentsandconfigs"
	authURL := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent&code_challenge=%s&code_challenge_method=S256",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
		url.QueryEscape(codeChallenge),
	)

	// 3. Open Browser
	fmt.Println()
	fmt.Println("Opening your default browser to sign in with Google / Antigravity...")
	fmt.Printf("If browser does not open automatically, visit this URL:\n\n%s\n\n", authURL)
	_ = openBrowser(authURL)

	// 4. Wait for callback
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errMsg := q.Get("error"); errMsg != "" {
			errChan <- fmt.Errorf("google oauth error: %s", errMsg)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("<html><body><h2>Login Failed</h2><p>" + errMsg + "</p></body></html>"))
			return
		}
		code := q.Get("code")
		if code == "" {
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>mnCode - Login Successful</title></head>
<body style="font-family:-apple-system,BlinkMacSystemFont,sans-serif;text-align:center;padding:60px 20px;background:#0d1117;color:#c9d1d9;">
  <h2 style="color:#58a6ff;">Login Successful!</h2>
  <p>You have successfully authorized mnCode with your Google / Antigravity account.</p>
  <p style="color:#8b949e;">You can now close this tab and return to your terminal.</p>
</body>
</html>`))
		codeChan <- code
	}

	mux.HandleFunc("/", handler)
	mux.HandleFunc("/callback", handler)

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	var code string
	select {
	case code = <-codeChan:
	case err := <-errChan:
		return nil, err
	case <-time.After(2 * time.Minute):
		return nil, fmt.Errorf("login timed out after 2 minutes")
	}

	// 5. Exchange Authorization Code for Tokens
	tokenData := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"code_verifier": {codeVerifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	}

	tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", tokenData)
	if err != nil {
		return nil, fmt.Errorf("failed exchanging token: %w", err)
	}
	defer tokenResp.Body.Close()

	body, _ := io.ReadAll(tokenResp.Body)
	if tokenResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", tokenResp.StatusCode, string(body))
	}

	var tokenResult struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tokenResult); err != nil {
		return nil, fmt.Errorf("failed parsing token response: %w", err)
	}

	// 6. Fetch User Info
	userEmail := "antigravity-user"
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tokenResult.AccessToken)
	if userResp, err := http.DefaultClient.Do(req); err == nil {
		defer userResp.Body.Close()
		if uBody, err := io.ReadAll(userResp.Body); err == nil {
			var uInfo GoogleUserInfo
			if err := json.Unmarshal(uBody, &uInfo); err == nil && uInfo.Email != "" {
				userEmail = uInfo.Email
			}
		}
	}

	// 7. Save Account in Store
	expiresAt := time.Time{}
	if tokenResult.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResult.ExpiresIn) * time.Second)
	}
	acc := &Account{
		ID:           userEmail,
		Email:        userEmail,
		Provider:     ProviderTypeAntigravity,
		AccessToken:  tokenResult.AccessToken,
		RefreshToken: tokenResult.RefreshToken,
		ExpiresAt:    expiresAt,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	if err := store.AddOrUpdate(acc); err != nil {
		return nil, err
	}

	return acc, nil
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
