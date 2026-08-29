package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

const maxReleaseManifestBytes = 4 << 20

var releaseHTTPClient = &http.Client{Timeout: 5 * time.Second}

func fetchReleaseManifest() (cliReleaseManifest, error) {
	endpoint, err := releaseFeedURL()
	if err != nil {
		return cliReleaseManifest{}, err
	}
	origin := endpointOrigin(endpoint)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return cliReleaseManifest{}, fmt.Errorf("create release request: %w", err)
	}
	req.Header.Set("User-Agent", "mncode-cli-updater")
	client := *releaseHTTPClient
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) >= maxReleaseRedirects {
			return fmt.Errorf("release endpoint redirected too many times")
		}
		if !sameURLOrigin(next.URL, origin) {
			return fmt.Errorf("release endpoint redirected to an untrusted origin")
		}
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return cliReleaseManifest{}, fmt.Errorf("fetch release manifest: %w", err)
	}
	defer resp.Body.Close()
	if resp.Request == nil || !sameURLOrigin(resp.Request.URL, origin) {
		return cliReleaseManifest{}, fmt.Errorf("release response came from an untrusted origin")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return cliReleaseManifest{}, fmt.Errorf("release endpoint returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReleaseManifestBytes+1))
	if err != nil {
		return cliReleaseManifest{}, fmt.Errorf("read release manifest: %w", err)
	}
	if len(data) > maxReleaseManifestBytes {
		return cliReleaseManifest{}, fmt.Errorf("release manifest exceeds size limit")
	}
	var manifest cliReleaseManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return cliReleaseManifest{}, fmt.Errorf("decode release manifest: %w", err)
	}
	if err := verifyReleaseManifest(manifest, time.Now().UTC()); err != nil {
		return cliReleaseManifest{}, err
	}
	if err := rememberReleaseManifest(manifest); err != nil {
		return cliReleaseManifest{}, err
	}
	return manifest, nil
}

func releaseFeedURL() (string, error) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("MNCODE_WEB_URL")), "/")
	if base == "" {
		base = releaseOrigin
	}
	parsed, err := url.Parse(base + releaseManifestEndpoint)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "https" && !isLocalHTTP(parsed)) {
		return "", fmt.Errorf("release feed must be an HTTPS URL without credentials, query, or fragment")
	}
	return parsed.String(), nil
}

func isLocalHTTP(parsed *url.URL) bool {
	host := parsed.Hostname()
	return parsed.Scheme == "http" && (strings.EqualFold(host, "localhost") || host == "127.0.0.1" || host == "::1")
}

func endpointOrigin(raw string) string {
	parsed, _ := url.Parse(raw)
	return parsed.Scheme + "://" + parsed.Host
}

func downloadReleaseAsset(asset cliReleaseAsset, executablePath string) error {
	parsed, err := url.Parse(asset.URL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || !sameURLOrigin(parsed, releaseAssetOrigin) {
		return fmt.Errorf("signed release asset %q has untrusted URL", asset.Name)
	}
	if asset.Size <= 0 || asset.Size > maxReleaseAssetSize {
		return fmt.Errorf("signed release asset %q has invalid size", asset.Name)
	}
	const pathPrefix = "/mncuchiinhuttt/mncode/releases/download/"
	parts := strings.Split(strings.TrimPrefix(parsed.Path, pathPrefix), "/")
	if !strings.HasPrefix(parsed.Path, pathPrefix) || len(parts) != 2 || parts[1] != asset.Name {
		return fmt.Errorf("signed release asset %q has untrusted path", asset.Name)
	}
	initialPath := parsed.Path
	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return fmt.Errorf("create release asset request: %w", err)
	}
	req.Header.Set("User-Agent", "mncode-cli-updater")
	client := *releaseHTTPClient
	client.Timeout = 30 * time.Second
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) >= maxReleaseRedirects {
			return fmt.Errorf("release asset redirected too many times")
		}
		if !trustedAssetRedirect(next.URL, initialPath) {
			return fmt.Errorf("release asset redirected to an untrusted origin")
		}
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download error: %w", err)
	}
	defer resp.Body.Close()
	if resp.Request == nil || !trustedAssetRedirect(resp.Request.URL, initialPath) {
		return fmt.Errorf("release asset response came from an untrusted origin")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("release asset download returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength >= 0 && resp.ContentLength != asset.Size {
		return fmt.Errorf("release asset size mismatch")
	}

	tmpPath := executablePath + ".tmp"
	completed := false
	defer func() {
		if !completed {
			_ = os.Remove(tmpPath)
		}
	}()
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("cannot create temp file for update: %w", err)
	}
	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(out, hasher), io.LimitReader(resp.Body, asset.Size+1))
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("error writing downloaded binary: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close downloaded binary: %w", closeErr)
	}
	if written != asset.Size {
		return fmt.Errorf("release asset size mismatch")
	}
	if got := hex.EncodeToString(hasher.Sum(nil)); !strings.EqualFold(got, asset.SHA256) {
		return fmt.Errorf("release asset sha256 mismatch")
	}
	if err := os.Rename(tmpPath, executablePath); err != nil {
		if runtime.GOOS != "windows" {
			return fmt.Errorf("cannot replace executable: %w", err)
		}
		_ = os.Rename(executablePath, executablePath+".old")
		if err = os.Rename(tmpPath, executablePath); err != nil {
			return fmt.Errorf("cannot replace executable (try with sudo or administrator): %w", err)
		}
	}
	completed = true
	return nil
}

func trustedAssetRedirect(raw *url.URL, initialPath string) bool {
	if raw == nil || raw.User != nil || raw.RawQuery != "" || raw.Fragment != "" || raw.Scheme != "https" {
		return false
	}
	if sameURLOrigin(raw, releaseAssetOrigin) {
		return raw.Path == initialPath
	}
	return sameURLOrigin(raw, releaseAssetCDNOrigin)
}
