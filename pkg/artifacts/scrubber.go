package artifacts

import "regexp"

type secretPattern struct {
	name    string
	pattern *regexp.Regexp
	replace string
}

var secretScrubbers = []secretPattern{
	{
		name:    "Anthropic API Key",
		pattern: regexp.MustCompile(`sk-ant-api03-[a-zA-Z0-9_-]{20,}`),
		replace: "[REDACTED_ANTHROPIC_KEY]",
	},
	{
		name:    "OpenAI API Key",
		pattern: regexp.MustCompile(`sk-(?:proj-)?[a-zA-Z0-9_-]{32,}`),
		replace: "[REDACTED_OPENAI_KEY]",
	},
	{
		name:    "Google OAuth Token",
		pattern: regexp.MustCompile(`ya29\.[a-zA-Z0-9_-]{20,}`),
		replace: "[REDACTED_GOOGLE_TOKEN]",
	},
	{
		name:    "GitHub Token",
		pattern: regexp.MustCompile(`gh[pousr]_[a-zA-Z0-9]{36}`),
		replace: "[REDACTED_GITHUB_TOKEN]",
	},
	{
		name:    "Brave Search Key",
		pattern: regexp.MustCompile(`BSA[a-zA-Z0-9_-]{20,}`),
		replace: "[REDACTED_BRAVE_KEY]",
	},
	{
		name:    "Tavily Search Key",
		pattern: regexp.MustCompile(`tvly-[a-zA-Z0-9_-]{20,}`),
		replace: "[REDACTED_TAVILY_KEY]",
	},
	{
		name:    "JWT Token",
		pattern: regexp.MustCompile(`eyJ[a-zA-Z0-9_-]{10,}\.eyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}`),
		replace: "[REDACTED_JWT_TOKEN]",
	},
	{
		name:    "Private Key Block",
		pattern: regexp.MustCompile(`(?s)-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----.*?-----END (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`),
		replace: "[REDACTED_PRIVATE_KEY_BLOCK]",
	},
}

// ScrubSecrets redacts sensitive credentials and tokens before storing artifacts.
func ScrubSecrets(content string) string {
	if content == "" {
		return ""
	}
	result := content
	for _, scrubber := range secretScrubbers {
		result = scrubber.pattern.ReplaceAllString(result, scrubber.replace)
	}
	return result
}
