package browserctl

import (
	"regexp"
	"strings"
)

var (
	passwordInputRe = regexp.MustCompile(`(?i)(<input[^>]*type=["']password["'][^>]*value=["'])([^"']*)(["'][^>]*>)`)
	authHeaderRe    = regexp.MustCompile(`(?i)(authorization:\s*bearer\s+)([a-zA-Z0-9_\-\.]+)`)
	creditCardRe    = regexp.MustCompile(`\b(?:\d{4}[ -]?){3}\d{4}\b`)
)

// MaskSensitiveDOM redacts password input values, auth tokens, and credit card numbers
// from DOM snapshots before sending them to LLM or telemetry.
func MaskSensitiveDOM(content string) string {
	if content == "" {
		return ""
	}

	result := passwordInputRe.ReplaceAllString(content, `${1}[REDACTED_PASSWORD]${3}`)
	result = authHeaderRe.ReplaceAllString(result, `${1}[REDACTED_AUTH_TOKEN]`)
	result = creditCardRe.ReplaceAllString(result, "[REDACTED_CREDIT_CARD]")

	return result
}

// IsConsequentialAction reports whether a browser action could have irreversible real-world side effects.
func IsConsequentialAction(action string, selectorOrTarget string) bool {
	act := strings.ToLower(action)
	if act != "click" && act != "submit" && act != "press" {
		return false
	}

	s := strings.ToLower(selectorOrTarget)
	dangerKeywords := []string{
		"pay", "checkout", "purchase", "buy", "order",
		"delete", "remove", "destroy", "drop", "purge",
		"transfer", "wire", "authorize",
	}

	for _, kw := range dangerKeywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
