package accounts

import (
	"fmt"
	"net/http"
	"time"
)

func checkOpenAIQuota(acc *Account, info *AccountQuotaInfo) {
	req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		info.Status = "Error"
		info.ErrorMessage = err.Error()
		return
	}
	req.Header.Set("Authorization", "Bearer "+acc.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		info.Status = "Unreachable"
		info.ErrorMessage = err.Error()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		info.Status = "Healthy [OK]"
		info.IsHealthy = true
		info.Tier = "OpenAI Tier"
		info.MaxContext = 128000
		info.AvailableModels = []string{"gpt-4o", "gpt-4o-mini", "o1", "o3-mini"}
		info.RPMRemaining = "500 RPM"
		info.TPMRemaining = "2,000,000 TPM"
	} else if resp.StatusCode == http.StatusTooManyRequests {
		info.Status = "Rate Limited [429]"
		info.ErrorMessage = "Quota exhausted or rate limit hit."
	} else if resp.StatusCode == http.StatusUnauthorized {
		info.Status = "Invalid API Key [401]"
	} else {
		info.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
}

func checkOpenCodeQuota(acc *Account, info *AccountQuotaInfo) {
	req, err := http.NewRequest("GET", "https://opencode.ai/zen/v1/models", nil)
	if err != nil {
		info.Status = "Error"
		info.ErrorMessage = err.Error()
		return
	}
	req.Header.Set("Authorization", "Bearer public")
	req.Header.Set("User-Agent", "opencode")
	req.Header.Set("x-opencode-client", "desktop")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		info.Status = "Unreachable"
		info.ErrorMessage = err.Error()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		info.Status = "Healthy [OK]"
		info.IsHealthy = true
		info.Tier = "OpenCode Zen Free Tier"
		info.MaxContext = 200000
		info.AvailableModels = []string{"deepseek-v4-flash-free", "x-preview-f-free", "mimo-v2.5-free", "nemotron-3-ultra-free"}
		info.RPMRemaining = "Free Tier"
		info.TPMRemaining = "Unlimited"
	} else {
		info.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
}
