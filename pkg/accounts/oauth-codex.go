package accounts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AddCodexAccount adds a Codex / OpenAI API token or session token
func AddCodexAccount(store *Store, emailOrID, token string) (*Account, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	id := emailOrID
	if id == "" {
		id = fmt.Sprintf("codex-%d", time.Now().Unix())
	}

	acc := &Account{
		ID:          id,
		Email:       emailOrID,
		Provider:    ProviderTypeCodex,
		AccessToken: token,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	if err := store.AddOrUpdate(acc); err != nil {
		return nil, err
	}

	return acc, nil
}

// ImportCodexCredentials scans local ~/.codex or OpenAI CLI configs if present
func ImportCodexCredentials(store *Store) (*Account, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(homeDir, ".openai", "credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var creds struct {
		APIKey string `json:"api_key"`
		Email  string `json:"email"`
	}
	if err := json.Unmarshal(data, &creds); err == nil && creds.APIKey != "" {
		return AddCodexAccount(store, creds.Email, creds.APIKey)
	}

	return nil, fmt.Errorf("no local codex credentials found")
}

// ValidateCodexToken tests if a token is valid by pinging the models API
func ValidateCodexToken(token string) (bool, error) {
	req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("validation failed (status %d): %s", resp.StatusCode, string(body))
}
