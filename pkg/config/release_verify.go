package config

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

func verifyReleaseManifest(manifest cliReleaseManifest, now time.Time) error {
	if manifest.SchemaVersion != releaseManifestSchemaVersion {
		return fmt.Errorf("release manifest schemaVersion %d is unsupported", manifest.SchemaVersion)
	}
	if !validReleaseVersion(manifest.Version) {
		return fmt.Errorf("release manifest has invalid version")
	}
	if manifest.Channel != "stable" && manifest.Channel != "beta" {
		return fmt.Errorf("release manifest has unsupported channel %q", manifest.Channel)
	}
	issued, err := time.Parse(time.RFC3339, manifest.IssuedAt)
	if err != nil {
		return fmt.Errorf("release manifest issuedAt is invalid: %w", err)
	}
	expires, err := time.Parse(time.RFC3339, manifest.ExpiresAt)
	if err != nil {
		return fmt.Errorf("release manifest expiresAt is invalid: %w", err)
	}
	if issued.After(expires) || expires.Sub(issued) > releaseManifestMaxAge || issued.After(now.Add(releaseClockSkew)) || !expires.After(now) {
		return fmt.Errorf("release manifest validity window is invalid or expired")
	}
	if manifest.KeyID != pinnedReleaseRootKeyID {
		return fmt.Errorf("release manifest keyID %q is not trusted", manifest.KeyID)
	}
	if len(manifest.Assets) == 0 {
		return fmt.Errorf("release manifest has no assets")
	}
	seen := make(map[string]struct{}, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		if err := validateReleaseAsset(manifest.Version, asset); err != nil {
			return err
		}
		if _, ok := seen[asset.Name]; ok {
			return fmt.Errorf("release manifest contains duplicate asset %q", asset.Name)
		}
		seen[asset.Name] = struct{}{}
	}
	signature, err := decodeReleaseSignature(manifest.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("release manifest signature is malformed")
	}
	payload, err := releaseManifestSigningBytes(manifest)
	if err != nil || !ed25519.Verify(pinnedReleaseKey, payload, signature) {
		return fmt.Errorf("release manifest signature mismatch for version %s", manifest.Version)
	}
	verifiedReleaseState.RLock()
	previous := verifiedReleaseState.latest[manifest.Channel]
	verifiedReleaseState.RUnlock()
	if previous != "" && compareVersion(manifest.Version, previous) < 0 {
		return fmt.Errorf("release manifest replay/rollback rejected: %s is older than %s", manifest.Version, previous)
	}
	return nil
}

func validateReleaseAsset(version string, asset cliReleaseAsset) error {
	if asset.Name == "" || asset.Name == "." || asset.Name == ".." || strings.ContainsAny(asset.Name, `/\\`) || path.Base(asset.Name) != asset.Name || strings.IndexFunc(asset.Name, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return fmt.Errorf("release manifest asset %q has unsafe name", asset.Name)
	}
	if asset.Size <= 0 || asset.Size > maxReleaseAssetSize {
		return fmt.Errorf("release manifest asset %q has invalid size", asset.Name)
	}
	if len(asset.SHA256) != sha256.Size*2 {
		return fmt.Errorf("release manifest asset %q has invalid sha256", asset.Name)
	}
	if _, err := hex.DecodeString(asset.SHA256); err != nil {
		return fmt.Errorf("release manifest asset %q has invalid sha256", asset.Name)
	}
	parsed, err := url.Parse(asset.URL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || !sameURLOrigin(parsed, releaseAssetOrigin) {
		return fmt.Errorf("release manifest asset %q has untrusted URL", asset.Name)
	}
	prefix := "/mncuchiinhuttt/mncode/releases/download/"
	parts := strings.Split(strings.TrimPrefix(parsed.Path, prefix), "/")
	if !strings.HasPrefix(parsed.Path, prefix) || len(parts) != 2 || parts[0] != version || parts[1] != asset.Name {
		return fmt.Errorf("release manifest asset %q is outside the trusted release path", asset.Name)
	}
	return nil
}

func compareVersion(a, b string) int {
	left, right := versionParts(a), versionParts(b)
	for i := 0; i < len(left) || i < len(right); i++ {
		l, r := 0, 0
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		if l != r {
			if l > r {
				return 1
			}
			return -1
		}
	}
	return 0
}

func newerVersion(current, latest string) bool {
	return compareVersion(latest, current) > 0
}

func versionParts(value string) []int {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	value = strings.SplitN(value, "-", 2)[0]
	parts := strings.Split(value, ".")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			number = 0
		}
		result = append(result, number)
	}
	return result
}

func validReleaseVersion(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "v") {
		return false
	}
	parts := strings.SplitN(strings.TrimPrefix(value, "v"), "-", 2)
	if len(parts) == 2 && (parts[1] == "" || strings.IndexFunc(parts[1], func(r rune) bool {
		return !(r == '.' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	}) >= 0) {
		return false
	}
	for _, part := range strings.Split(parts[0], ".") {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}
