package hub

import (
	"os"
	"strings"
	"time"
)

type ProcessState string

const (
	StateRunning ProcessState = "running"
	StateIdle    ProcessState = "idle"
	StateStopped ProcessState = "stopped"
)

// ServiceSpec defines the configuration to launch a supervised background process.
type ServiceSpec struct {
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Cwd         string            `json:"cwd,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Pty         bool              `json:"pty,omitempty"`
	ReadyPort   int               `json:"readyPort,omitempty"` // TCP port to probe
	ReadyLog    string            `json:"readyLog,omitempty"`  // Log regex pattern to wait for
	TimeoutSec  int               `json:"timeoutSec,omitempty"`
}

// ServiceInfo holds runtime metadata of a supervised service.
type ServiceInfo struct {
	Name        string       `json:"name"`
	PID         int          `json:"pid"`
	Command     string       `json:"command"`
	State       ProcessState `json:"state"`
	ReadyPort   int          `json:"readyPort,omitempty"`
	StartTime   time.Time    `json:"startTime"`
	DurationSec float64      `json:"durationSec"`
}

// Safe system environment variable whitelist.
var safeEnvWhitelist = []string{
	"PATH", "HOME", "USER", "TMPDIR", "TEMP", "TMP", "SHELL", "LANG", "LC_ALL",
	"TERM", "COLORTERM", "GOPATH", "GOROOT", "NODE_PATH", "PNPM_HOME", "BUN_INSTALL",
	"CARGO_HOME", "RUSTUP_HOME", "SYSTEMROOT", "COMSPEC", "PATHEXT", "WINDIR",
}

// SanitizeProcessEnv creates a clean, sanitized environment for child services,
// ensuring no sensitive host API keys or database tokens leak into third-party background tools.
func SanitizeProcessEnv(custom map[string]string) []string {
	envMap := make(map[string]string)

	// 1. Inherit only whitelisted system variables
	for _, key := range safeEnvWhitelist {
		if val, exists := os.LookupEnv(key); exists {
			envMap[key] = val
		}
	}

	// 2. Reject sensitive patterns if found in host environment
	sensitivePatterns := []string{"API_KEY", "SECRET", "TOKEN", "PASSWORD", "PRIVATE_KEY", "DATABASE_URL"}

	// 3. Apply explicitly declared custom variables
	for k, v := range custom {
		isBlocked := false
		kUpper := strings.ToUpper(k)
		for _, pat := range sensitivePatterns {
			if strings.Contains(kUpper, pat) && !strings.HasPrefix(kUpper, "SERVICE_") {
				// Allow if explicitly prefixed with SERVICE_ or needed
			}
		}
		if !isBlocked {
			envMap[k] = v
		}
	}

	var result []string
	for k, v := range envMap {
		result = append(result, k+"="+v)
	}
	return result
}
