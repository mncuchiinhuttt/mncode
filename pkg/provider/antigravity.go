package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type AntigravityProvider struct {
	AccessToken      string
	RefreshToken     string
	BaseURL          string
	ProjectID        string
	HTTPClient       *http.Client
	OnTokenRefreshed func(newTok string)
	accountID        string
}

func (a *AntigravityProvider) AccountID() string      { return a.accountID }
func (a *AntigravityProvider) SetAccountID(id string) { a.accountID = id }

func NewAntigravityProvider(accessToken, baseURL string) *AntigravityProvider {
	if baseURL == "" || !strings.Contains(baseURL, "googleapis.com") {
		baseURL = "https://daily-cloudcode-pa.googleapis.com/v1internal"
	}
	return &AntigravityProvider{
		AccessToken: accessToken,
		BaseURL:     baseURL,
		HTTPClient:  &http.Client{Timeout: 10 * time.Minute},
	}
}

func (a *AntigravityProvider) Name() string { return "antigravity" }

func mapAntigravityModel(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	switch m {
	case "gemini-3.7-flash-high", "gemini-3.7-flash", "gemini-flash", "gemini-2.5-flash", "":
		return "gemini-3.6-flash-high"
	case "gemini-2.5-pro", "gemini-3-pro", "gemini-3.1-pro", "gemini-pro":
		return "gemini-pro-agent"
	case "claude-3-7-sonnet", "claude-3.7-sonnet", "claude-sonnet", "claude-3-5-sonnet", "claude":
		return "claude-sonnet-4-6"
	case "claude-3-opus", "claude-opus":
		return "claude-opus-4-6-thinking"
	case "gpt-4o", "gpt-4", "gpt-oss":
		return "gpt-oss-120b-medium"
	default:
		return model
	}
}

func (a *AntigravityProvider) RefreshTokenNow() (string, error) {
	if a.RefreshToken == "" {
		return "", fmt.Errorf("no refresh token available")
	}
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		clientID = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	}
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = "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"
	}

	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {a.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body := []byte(readProviderErrorBody(resp.Body))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed refreshing token: %s", string(body))
	}

	var res struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &res); err != nil || res.AccessToken == "" {
		return "", fmt.Errorf("invalid token refresh response: %s", string(body))
	}

	a.AccessToken = res.AccessToken
	if a.OnTokenRefreshed != nil {
		a.OnTokenRefreshed(res.AccessToken)
	}
	return res.AccessToken, nil
}

func (a *AntigravityProvider) EnsureProjectID(ctx context.Context) string {
	if a.ProjectID != "" {
		return a.ProjectID
	}
	loadReqBody := []byte(`{"metadata":{"ideType":9,"platform":2,"pluginType":2},"mode":1}`)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist", bytes.NewReader(loadReqBody))
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+a.AccessToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1 vscode-antigravity/1.107.0")
		if resp, err := a.HTTPClient.Do(req); err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var codeAssist struct {
					ProjectID string `json:"cloudaicompanionProject"`
				}
				body := []byte(readProviderErrorBody(resp.Body))
				_ = json.Unmarshal(body, &codeAssist)
				if codeAssist.ProjectID != "" {
					a.ProjectID = codeAssist.ProjectID
					return a.ProjectID
				}
			}
		}
	}
	return ""
}

func (a *AntigravityProvider) Stream(ctx context.Context, req *CompletionRequest, cb func(StreamEvent) error) (*CompletionResponse, error) {
	projectID := a.EnsureProjectID(ctx)
	targetModel := mapAntigravityModel(req.Model)
	url := fmt.Sprintf("%s:streamGenerateContent?alt=sse", a.BaseURL)

	geminiPayload := a.buildGeminiRequest(req)
	wrapperPayload := map[string]interface{}{
		"model":   targetModel,
		"request": geminiPayload,
	}
	if projectID != "" {
		wrapperPayload["project"] = projectID
	}

	jsonBody, err := json.Marshal(wrapperPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal antigravity payload: %w", err)
	}

	makeReq := func(token, targetURL string) (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		tok := token
		if !strings.HasPrefix(tok, "Bearer ") {
			tok = "Bearer " + tok
		}
		httpReq.Header.Set("Authorization", tok)
		httpReq.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1 vscode-antigravity/1.107.0")
		httpReq.Header.Set("X-Client-Name", "antigravity")
		httpReq.Header.Set("X-Client-Version", "1.107.0")
		httpReq.Header.Set("X-Goog-Api-Client", "antigravity/1.107.0")
		return a.HTTPClient.Do(httpReq)
	}
	resp, err := makeReq(a.AccessToken, url)
	if err != nil {
		return nil, newNetworkError(a.Name(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes := []byte(readProviderErrorBody(resp.Body))
		message := string(bodyBytes)
		if resp.StatusCode == http.StatusTooManyRequests {
			message = fmt.Sprintf("Quota exhausted for model '%s'.\n\n[Tip] Switch model with '/model' to switch account with '/account'.", targetModel)
		} else {
			message += "\n\n[Tip] Run '/login antigravity' to re-authenticate or '/account import' to refresh credentials."
		}
		return nil, newHTTPError(a.Name(), resp.StatusCode, message, resp.Header)
	}

	return a.parseSSE(resp.Body, cb)
}
