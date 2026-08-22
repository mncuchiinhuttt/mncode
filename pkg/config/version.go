package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

const (
	CurrentVersion = "v0.1.2.7-beta"
	GithubRepo     = "mncuchiinhuttt/mncode"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// CheckLatestVersion queries GitHub Releases for available updates
func CheckLatestVersion() (latestTag string, hasUpdate bool, err error) {
	client := &http.Client{Timeout: 3 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", GithubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("User-Agent", "mncode-cli-updater")

	resp, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", false, err
	}

	if len(releases) == 0 {
		return CurrentVersion, false, nil
	}

	latestTag = releases[0].TagName
	if latestTag != "" && latestTag != CurrentVersion {
		return latestTag, true, nil
	}

	return latestTag, false, nil
}

// PerformSelfUpdate downloads and replaces the currently running binary
func PerformSelfUpdate(targetTag string) error {
	if targetTag == "" {
		tag, hasUpdate, err := CheckLatestVersion()
		if err != nil {
			return err
		}
		if !hasUpdate {
			return fmt.Errorf("already running the latest version (%s)", CurrentVersion)
		}
		targetTag = tag
	}

	osName := runtime.GOOS
	archName := runtime.GOARCH

	binaryExt := ""
	if osName == "windows" {
		binaryExt = ".exe"
	}

	assetName := fmt.Sprintf("mncode-%s-%s%s", osName, archName, binaryExt)
	downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", GithubRepo, targetTag, assetName)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("release asset download returned HTTP %d for %s", resp.StatusCode, downloadURL)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find current executable path: %w", err)
	}

	tmpFile := execPath + ".tmp"
	out, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("cannot create temp file for update: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("error writing downloaded binary: %w", err)
	}

	// Atomic replace
	err = os.Rename(tmpFile, execPath)
	if err != nil {
		// Fallback for Windows in-use binary locking
		_ = os.Rename(execPath, execPath+".old")
		err = os.Rename(tmpFile, execPath)
		if err != nil {
			return fmt.Errorf("cannot replace executable (try with sudo or administrator): %w", err)
		}
	}

	return nil
}
