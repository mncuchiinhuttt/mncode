package accounts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ImportAntigravityDefaultCreds tries to import credentials from local ~/.gemini or ~/.antigravity
func ImportAntigravityDefaultCreds(store *Store) (*Account, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	paths := []string{
		filepath.Join(homeDir, ".gemini", "jetski-standalone-oauth-token"),
		filepath.Join(homeDir, ".gemini", "antigravity-cli", "credentials.json"),
		filepath.Join(homeDir, ".gemini", "credentials.json"),
		filepath.Join(homeDir, ".antigravity", "credentials.json"),
	}

	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			var creds struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				Email        string `json:"email"`
				Token        struct {
					AccessToken  string `json:"access_token"`
					RefreshToken string `json:"refresh_token"`
					Expiry       string `json:"expiry"`
				} `json:"token"`
			}
			if err := json.Unmarshal(data, &creds); err == nil {
				accessToken := creds.AccessToken
				refreshToken := creds.RefreshToken
				if accessToken == "" {
					accessToken = creds.Token.AccessToken
				}
				if refreshToken == "" {
					refreshToken = creds.Token.RefreshToken
				}
				expiresAt := parseCredentialExpiry(creds.Token.Expiry)
				if accessToken != "" || refreshToken != "" {
					if refreshToken != "" {
						if newTok, err := RefreshGoogleToken(refreshToken, "", ""); err == nil && newTok != "" {
							accessToken = newTok
							expiresAt = time.Now().Add(time.Hour)
						}
					}
					email := creds.Email
					if email == "" {
						email = "antigravity-default"
					}
					acc := &Account{
						ID:           email,
						Email:        email,
						Provider:     ProviderTypeAntigravity,
						AccessToken:  accessToken,
						RefreshToken: refreshToken,
						ExpiresAt:    expiresAt,
						IsActive:     true,
						CreatedAt:    time.Now(),
					}
					_ = store.AddOrUpdate(acc)
					return acc, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no local antigravity credentials found")
}

func parseCredentialExpiry(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if seconds > 1_000_000_000_000 {
			return time.UnixMilli(seconds)
		}
		return time.Unix(seconds, 0)
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if value, err := time.Parse(layout, raw); err == nil {
			return value
		}
	}
	return time.Time{}
}

// RefreshGoogleToken refreshes Google OAuth tokens using a refresh token
func RefreshGoogleToken(refreshToken, clientID, clientSecret string) (string, error) {
	if clientID == "" {
		clientID = os.Getenv("GOOGLE_CLIENT_ID")
		if clientID == "" {
			clientID = defaultAntigravityClientID
		}
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
		if clientSecret == "" {
			clientSecret = defaultAntigravityClientSecret
		}
	}

	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed refreshing token: %s", string(body))
	}

	var res struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", err
	}

	return res.AccessToken, nil
}
