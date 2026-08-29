package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	releaseManifestEndpoint      = "/api/releases/latest"
	releaseOrigin                = "https://mncode.mncuchiinhuttt.dev"
	releaseManifestSchemaVersion = 1
	releaseAssetOrigin           = "https://github.com"
	releaseAssetCDNOrigin        = "https://release-assets.githubusercontent.com"
	releaseClockSkew             = 5 * time.Minute
	releaseManifestMaxAge        = 31 * 24 * time.Hour
	maxReleaseAssetSize          = 256 << 20
	maxReleaseRedirects          = 3
	pinnedReleaseRootKeyID       = "mncode-release-2026"
	pinnedReleaseRootKeyBase64   = "Ln/WP/P6PGgmNpwoO0waNBGllXtafV6YLels+cj7yMQ="
)

type cliReleaseAsset struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type cliReleaseManifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Version       string            `json:"version"`
	Channel       string            `json:"channel"`
	IssuedAt      string            `json:"issuedAt"`
	ExpiresAt     string            `json:"expiresAt"`
	KeyID         string            `json:"keyID"`
	Signature     string            `json:"signature"`
	Assets        []cliReleaseAsset `json:"assets"`
}

type cliManifestPayload struct {
	SchemaVersion int               `json:"schemaVersion"`
	Version       string            `json:"version"`
	Channel       string            `json:"channel"`
	IssuedAt      string            `json:"issuedAt"`
	ExpiresAt     string            `json:"expiresAt"`
	Assets        []cliReleaseAsset `json:"assets"`
}

var (
	pinnedReleaseKey     = mustReleasePublicKey(pinnedReleaseRootKeyBase64)
	verifiedReleaseState = struct {
		sync.RWMutex
		latest map[string]string
	}{latest: make(map[string]string)}
)

func mustReleasePublicKey(encoded string) ed25519.PublicKey {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != ed25519.PublicKeySize {
		panic("invalid pinned release public key")
	}
	return ed25519.PublicKey(key)
}

func releaseManifestSigningBytes(manifest cliReleaseManifest) ([]byte, error) {
	assets := append([]cliReleaseAsset(nil), manifest.Assets...)
	for i := range assets {
		assets[i].SHA256 = strings.ToLower(assets[i].SHA256)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Name < assets[j].Name })
	return json.Marshal(cliManifestPayload{manifest.SchemaVersion, manifest.Version, manifest.Channel, manifest.IssuedAt, manifest.ExpiresAt, assets})
}

func decodeReleaseSignature(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	return hex.DecodeString(value)
}

func rememberReleaseManifest(manifest cliReleaseManifest) error {
	verifiedReleaseState.Lock()
	defer verifiedReleaseState.Unlock()
	previous := verifiedReleaseState.latest[manifest.Channel]
	if previous != "" && newerVersion(manifest.Version, previous) {
		return fmt.Errorf("release manifest replay/rollback rejected: %s follows newer %s", manifest.Version, previous)
	}
	verifiedReleaseState.latest[manifest.Channel] = manifest.Version
	return nil
}

func selectReleaseAsset(manifest cliReleaseManifest, name string) (cliReleaseAsset, error) {
	for _, asset := range manifest.Assets {
		if asset.Name == name {
			return asset, nil
		}
	}
	return cliReleaseAsset{}, fmt.Errorf("signed release has no asset for %s", name)
}

func sameURLOrigin(raw *url.URL, expected string) bool {
	want, err := url.Parse(expected)
	return err == nil && raw != nil && raw.Scheme == want.Scheme && strings.EqualFold(raw.Host, want.Host)
}
