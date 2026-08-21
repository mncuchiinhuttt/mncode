package accounts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

func checkAntigravityQuota(store *Store, acc *Account, info *AccountQuotaInfo) {
	token := acc.AccessToken
	client := &http.Client{Timeout: 12 * time.Second}

	// 1. Check token validity / expiry
	tokenInfoURL := fmt.Sprintf("https://www.googleapis.com/oauth2/v2/tokeninfo?access_token=%s", token)
	if tResp, err := client.Get(tokenInfoURL); err == nil {
		defer tResp.Body.Close()
		if tResp.StatusCode == http.StatusOK {
			var tInfo struct {
				Email     string `json:"email"`
				ExpiresIn int    `json:"expires_in"`
			}
			if b, _ := io.ReadAll(tResp.Body); json.Unmarshal(b, &tInfo) == nil {
				if tInfo.Email != "" {
					info.AccountID = tInfo.Email
				}
				if tInfo.ExpiresIn > 0 {
					info.ExpiresInStr = fmt.Sprintf("%dm remaining", tInfo.ExpiresIn/60)
				}
			}
		} else if (tResp.StatusCode == http.StatusBadRequest || tResp.StatusCode == http.StatusUnauthorized) && acc.RefreshToken != "" {
			if newToken, rErr := RefreshGoogleToken(acc.RefreshToken, defaultAntigravityClientID, defaultAntigravityClientSecret); rErr == nil {
				token = newToken
				acc.AccessToken = newToken
				_ = store.AddOrUpdate(acc)
				info.ExpiresInStr = "59m remaining (refreshed)"
			}
		}
	}

	// 2. Call loadCodeAssist to get Subscription Tier and Companion Project
	loadReqBody := []byte(`{"metadata":{"ideType":9,"platform":2,"pluginType":2},"mode":1}`)
	req, err := http.NewRequest("POST", "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist", bytes.NewReader(loadReqBody))
	if err != nil {
		info.Status = "Error"
		info.ErrorMessage = err.Error()
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1 vscode-antigravity/1.107.0")

	resp, err := client.Do(req)
	if err != nil {
		info.Status = "Unreachable"
		info.ErrorMessage = err.Error()
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized && acc.RefreshToken != "" {
		if newToken, rErr := RefreshGoogleToken(acc.RefreshToken, defaultAntigravityClientID, defaultAntigravityClientSecret); rErr == nil {
			token = newToken
			acc.AccessToken = newToken
			_ = store.AddOrUpdate(acc)
			req.Header.Set("Authorization", "Bearer "+token)
			if rResp, rErr := client.Do(req); rErr == nil {
				resp = rResp
				defer resp.Body.Close()
				body, _ = io.ReadAll(resp.Body)
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			info.Status = "Expired [401]"
			info.ErrorMessage = "Token expired. Run /login antigravity to re-authenticate."
		} else {
			info.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
			info.ErrorMessage = string(body)
		}
		return
	}

	var codeAssist struct {
		CurrentTier struct {
			Name string `json:"name"`
		} `json:"currentTier"`
		PaidTier struct {
			Name string `json:"name"`
		} `json:"paidTier"`
		ProjectID string `json:"cloudaicompanionProject"`
	}
	_ = json.Unmarshal(body, &codeAssist)

	info.Status = "Healthy [OK]"
	info.IsHealthy = true
	info.ProjectID = codeAssist.ProjectID

	tierName := codeAssist.CurrentTier.Name
	if tierName == "" {
		tierName = "Antigravity Early Access"
	}
	if codeAssist.PaidTier.Name != "" && codeAssist.PaidTier.Name != tierName {
		tierName = fmt.Sprintf("%s (%s)", tierName, codeAssist.PaidTier.Name)
	}
	info.Tier = tierName
	info.MaxContext = 1000000
	info.RPMRemaining = "360 RPM"
	info.TPMRemaining = "4,000,000 TPM"

	// 3. Call fetchAvailableModels to get Per-Model Quotas
	fetchBody, _ := json.Marshal(map[string]string{"project": codeAssist.ProjectID})
	mReq, err := http.NewRequest("POST", "https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels", bytes.NewReader(fetchBody))
	if err == nil {
		mReq.Header.Set("Authorization", "Bearer "+token)
		mReq.Header.Set("Content-Type", "application/json")
		mReq.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1 vscode-antigravity/1.107.0")
		mReq.Header.Set("X-Client-Name", "antigravity")
		mReq.Header.Set("X-Client-Version", "1.107.0")

		if mResp, mErr := client.Do(mReq); mErr == nil {
			defer mResp.Body.Close()
			if mResp.StatusCode == http.StatusOK {
				var mData struct {
					Models map[string]struct {
						DisplayName string `json:"displayName"`
						QuotaInfo   *struct {
							RemainingFraction float64 `json:"remainingFraction"`
							ResetTime         string  `json:"resetTime"`
						} `json:"quotaInfo"`
					} `json:"models"`
				}
				if mBytes, err := io.ReadAll(mResp.Body); err == nil && json.Unmarshal(mBytes, &mData) == nil {
					parseModelQuotas(mData.Models, info)
				}
			}
		}
	}
}

func parseModelQuotas(rawModels map[string]struct {
	DisplayName string `json:"displayName"`
	QuotaInfo   *struct {
		RemainingFraction float64 `json:"remainingFraction"`
		ResetTime         string  `json:"resetTime"`
	} `json:"quotaInfo"`
}, info *AccountQuotaInfo) {
	priorityOrder := map[string]int{
		"claude-sonnet-4-6":        1,
		"claude-opus-4-6-thinking": 2,
		"gemini-3.7-flash-high":    3,
		"gemini-3.6-flash-high":    4,
		"gemini-pro-agent":         5,
		"gpt-oss-120b-medium":      6,
		"gemini-3.1-flash-image":   7,
		"gemini-2.5-flash-lite":    8,
	}

	modelNames := map[string]string{
		"claude-sonnet-4-6":        "Claude Sonnet 4.6 (Thinking)",
		"claude-opus-4-6-thinking": "Claude Opus 4.6 (Thinking)",
		"gemini-3.7-flash-high":    "Gemini 3.7 Flash (High)",
		"gemini-3.6-flash-high":    "Gemini 3.6 Flash (High)",
		"gemini-pro-agent":         "Gemini 3.1 Pro (High)",
		"gpt-oss-120b-medium":      "GPT-OSS 120B (Medium)",
		"gemini-3.1-flash-image":   "Gemini 3.1 Flash (Image)",
		"gemini-2.5-flash-lite":    "Gemini 3.1 Flash Lite",
	}

	for id, m := range rawModels {
		if m.QuotaInfo == nil {
			continue
		}
		if _, ok := priorityOrder[id]; !ok {
			continue
		}
		name := m.DisplayName
		if name == "" || strings.HasPrefix(name, "MODEL_") {
			if friendly, ok := modelNames[id]; ok {
				name = friendly
			} else {
				name = id
			}
		}
		if friendly, ok := modelNames[id]; ok {
			name = friendly
		}

		remPct := m.QuotaInfo.RemainingFraction * 100.0
		resetIn := ""
		if m.QuotaInfo.ResetTime != "" {
			if t, err := time.Parse(time.RFC3339, m.QuotaInfo.ResetTime); err == nil {
				diff := time.Until(t)
				if diff > 0 {
					h := int(diff.Hours())
					m := int(diff.Minutes()) % 60
					if h > 0 {
						resetIn = fmt.Sprintf("resets in %dh %dm", h, m)
					} else {
						resetIn = fmt.Sprintf("resets in %dm", m)
					}
				}
			}
		}

		info.ModelQuotas = append(info.ModelQuotas, ModelQuota{
			ModelID:             id,
			DisplayName:         name,
			RemainingPercentage: remPct,
			ResetTime:           m.QuotaInfo.ResetTime,
			ResetInStr:          resetIn,
		})
		info.AvailableModels = append(info.AvailableModels, id)
	}

	sort.Slice(info.ModelQuotas, func(i, j int) bool {
		pI := 999
		if p, ok := priorityOrder[info.ModelQuotas[i].ModelID]; ok {
			pI = p
		}
		pJ := 999
		if p, ok := priorityOrder[info.ModelQuotas[j].ModelID]; ok {
			pJ = p
		}
		if pI != pJ {
			return pI < pJ
		}
		return info.ModelQuotas[i].ModelID < info.ModelQuotas[j].ModelID
	})
}
