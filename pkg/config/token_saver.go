// Token-saving directives and proxy management shared by the CLI and Desktop.
// Directives are injected at prompt-build time — the user's stored custom
// instructions are never modified.
package config

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const HeadroomProxyURL = "http://127.0.0.1:8787"

// RTKInstalled reports whether the rtk CLI is available (PATH or common
// install locations).
func RTKInstalled() bool {
	if _, err := exec.LookPath("rtk"); err == nil {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	candidates := []string{
		filepath.Join(home, ".local", "bin", "rtk"),
		"/opt/homebrew/bin/rtk",
		"/usr/local/bin/rtk",
	}
	if runtime.GOOS == "windows" {
		candidates = []string{
			filepath.Join(home, ".local", "bin", "rtk.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "rtk", "rtk.exe"),
		}
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}

// HeadroomInstalled reports whether the headroom CLI is on PATH.
func HeadroomInstalled() bool {
	_, err := exec.LookPath("headroom")
	return err == nil
}

func headroomProxyRunning() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:8787", 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// EnsureHeadroomProxy starts the local headroom proxy when installed and not
// already listening. Returns the proxy base URL, or "" when unavailable.
func EnsureHeadroomProxy() string {
	if headroomProxyRunning() {
		return HeadroomProxyURL
	}
	headroom, err := exec.LookPath("headroom")
	if err != nil {
		return ""
	}
	command := exec.Command(headroom, "proxy", "--port", "8787")
	command.SysProcAttr = detachedSysProcAttr()
	if err := command.Start(); err != nil {
		return ""
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if headroomProxyRunning() {
			return HeadroomProxyURL
		}
		time.Sleep(250 * time.Millisecond)
	}
	return ""
}

// TokenSaverDirectives returns the instruction block for the enabled token
// savers.
func TokenSaverDirectives(cfg *Config) []string {
	directives := []string{}
	if cfg.GetSetting("token_saver_concise", "false") == "true" {
		directives = append(directives,
			"Token-saving mode: keep responses concise and direct. Summarize instead of quoting large blocks. Never repeat file contents you have already referenced.")
	}
	if cfg.GetSetting("token_saver_compress_output", "false") == "true" {
		directives = append(directives,
			"Shell output discipline: when a command may produce long output, pipe it through `head -100`, `tail -50`, or `grep` filters. Never cat or print entire files; read the specific ranges you need.")
	}
	if cfg.GetSetting("token_saver_targeted_edits", "false") == "true" {
		directives = append(directives,
			"Editing discipline: prefer search-and-replace edits (replace_file_content) over rewriting whole files, and read only the specific line ranges relevant to the change.")
	}
	if cfg.GetSetting("token_saver_rtk", "false") == "true" && RTKInstalled() {
		directives = append(directives,
			"Shell output compression: the `rtk` CLI is installed on this machine. Prefix common development commands with `rtk` (for example `rtk git log --oneline -20`, `rtk npm test`, `rtk go build ./...`) so their output is token-compressed. If an rtk-wrapped command fails, fall back to the raw command.")
	}
	return directives
}
