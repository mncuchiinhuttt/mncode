package codex

import (
	"context"
	"fmt"
	"time"
)

// LoginBrowserFlow initiates official browser login and returns the auth URL.
func LoginBrowserFlow(ctx context.Context, client *Client) (*LoginStartResult, error) {
	if client == nil {
		return nil, fmt.Errorf("codex client is required")
	}

	var res LoginStartResult
	err := client.Call(ctx, "account/login/start", LoginStartParams{Type: "chatgpt"}, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to start chatgpt login: %w", err)
	}

	if res.AuthURL == "" {
		return nil, fmt.Errorf("no auth URL returned by app-server")
	}

	return &res, nil
}

// LoginDeviceCodeFlow initiates headless device-code login for servers/terminals.
func LoginDeviceCodeFlow(ctx context.Context, client *Client) (*LoginStartResult, error) {
	if client == nil {
		return nil, fmt.Errorf("codex client is required")
	}

	var res LoginStartResult
	err := client.Call(ctx, "account/login/start", LoginStartParams{Type: "chatgptDeviceCode"}, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to start device-code login: %w", err)
	}

	return &res, nil
}

// WaitForLoginComplete polls or listens for login completion until timeout.
func WaitForLoginComplete(ctx context.Context, client *Client, timeout time.Duration) (*AccountInfo, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("login timed out after %v", timeout)
			}
			account, err := ReadAccount(ctx, client)
			if err == nil && account != nil && account.AccountID != "" {
				return account, nil
			}
		}
	}
}

// ReadAccount queries active account profile.
func ReadAccount(ctx context.Context, client *Client) (*AccountInfo, error) {
	if client == nil {
		return nil, fmt.Errorf("codex client is required")
	}

	var res AccountReadResult
	err := client.Call(ctx, "account/read", nil, &res)
	if err != nil {
		return nil, err
	}
	return res.Account, nil
}

// Logout signs out of ChatGPT in the app-server.
func Logout(ctx context.Context, client *Client) error {
	if client == nil {
		return fmt.Errorf("codex client is required")
	}
	return client.Call(ctx, "account/logout", nil, nil)
}
