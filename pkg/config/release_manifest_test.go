package config

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func signedManifestFixture(t *testing.T, version string, asset cliReleaseAsset) cliReleaseManifest {
	t.Helper()
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	pinnedReleaseKey = public
	now := time.Now().UTC().Truncate(time.Second)
	manifest := cliReleaseManifest{1, version, "stable", now.Add(-time.Minute).Format(time.RFC3339), now.Add(time.Hour).Format(time.RFC3339), pinnedReleaseRootKeyID, "", []cliReleaseAsset{asset}}
	payload, err := releaseManifestSigningBytes(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(private, payload))
	return manifest
}

func resetReleaseState() {
	verifiedReleaseState.Lock()
	verifiedReleaseState.latest = make(map[string]string)
	verifiedReleaseState.Unlock()
}

func TestVerifySignedManifestAndSelectAsset(t *testing.T) {
	originalKey := pinnedReleaseKey
	defer func() { pinnedReleaseKey = originalKey; resetReleaseState() }()
	data := []byte("signed cli binary")
	sum := sha256.Sum256(data)
	asset := cliReleaseAsset{"mncode-darwin-arm64", "https://github.com/mncuchiinhuttt/mncode/releases/download/v9.2.0/mncode-darwin-arm64", int64(len(data)), hex.EncodeToString(sum[:])}
	manifest := signedManifestFixture(t, "v9.2.0", asset)
	if err := verifyReleaseManifest(manifest, time.Now().UTC()); err != nil {
		t.Fatalf("verifyReleaseManifest() error = %v", err)
	}
	selected, err := selectReleaseAsset(manifest, "mncode-darwin-arm64")
	if err != nil {
		t.Fatal(err)
	}
	if selected.URL != asset.URL {
		t.Fatalf("selected URL = %q, want %q", selected.URL, asset.URL)
	}
}

func TestManifestReplayAndOriginAreRejected(t *testing.T) {
	originalKey := pinnedReleaseKey
	defer func() { pinnedReleaseKey = originalKey; resetReleaseState() }()
	asset := cliReleaseAsset{"mncode-linux-amd64", "https://github.com/mncuchiinhuttt/mncode/releases/download/v9.2.0/mncode-linux-amd64", 3, strings.Repeat("a", 64)}
	newer := signedManifestFixture(t, "v9.2.0", asset)
	if err := verifyReleaseManifest(newer, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	rememberReleaseManifest(newer)
	olderAsset := asset
	olderAsset.URL = strings.Replace(olderAsset.URL, "v9.2.0", "v9.1.0", 1)
	older := signedManifestFixture(t, "v9.1.0", olderAsset)
	if err := verifyReleaseManifest(older, time.Now().UTC()); err == nil || !strings.Contains(err.Error(), "replay") {
		t.Fatalf("rollback error = %v", err)
	}

	bad := signedManifestFixture(t, "v9.3.0", asset)
	bad.Assets[0].URL = "https://evil.example/mncode-linux-amd64"
	if err := verifyReleaseManifest(bad, time.Now().UTC()); err == nil || !strings.Contains(err.Error(), "untrusted") {
		t.Fatalf("origin error = %v", err)
	}
}

type releaseRoundTripper struct{ body []byte }

func (r releaseRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Request: req, Body: io.NopCloser(bytes.NewReader(r.body)), ContentLength: int64(len(r.body)), Header: make(http.Header)}, nil
}

func TestDownloadAssetHashFailureDoesNotReplace(t *testing.T) {
	oldClient := releaseHTTPClient
	defer func() { releaseHTTPClient = oldClient }()
	releaseHTTPClient = &http.Client{Transport: releaseRoundTripper{body: []byte("tampered")}}
	target := filepath.Join(t.TempDir(), "mncode")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	asset := cliReleaseAsset{"mncode-linux-amd64", "https://github.com/mncuchiinhuttt/mncode/releases/download/v9.2.0/mncode-linux-amd64", 8, strings.Repeat("0", 64)}
	if err := downloadReleaseAsset(asset, target); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("hash error = %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old binary" {
		t.Fatalf("target was replaced with %q", got)
	}
	if _, err := os.Stat(target + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file still exists: %v", err)
	}
}
