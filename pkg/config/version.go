package config

import (
	"fmt"
	"os"
	"runtime"
)

const (
	CurrentVersion = "v0.1.6-beta"
	GithubRepo     = "mncuchiinhuttt/mncode"
)

// CheckLatestVersion queries and verifies the signed CLI release feed.
func CheckLatestVersion() (latestTag string, hasUpdate bool, err error) {
	manifest, err := fetchReleaseManifest()
	if err != nil {
		return "", false, err
	}
	if !newerVersion(CurrentVersion, manifest.Version) {
		return manifest.Version, false, nil
	}
	return manifest.Version, true, nil
}

// PerformSelfUpdate downloads and replaces the currently running binary.
func PerformSelfUpdate(targetTag string) error {
	manifest, err := fetchReleaseManifest()
	if err != nil {
		return err
	}
	if targetTag == "" {
		if !newerVersion(CurrentVersion, manifest.Version) {
			return fmt.Errorf("already running the latest version (%s)", CurrentVersion)
		}
		targetTag = manifest.Version
	} else if targetTag != manifest.Version {
		return fmt.Errorf("requested version %q does not match signed release %q", targetTag, manifest.Version)
	}
	if !newerVersion(CurrentVersion, targetTag) {
		return fmt.Errorf("already running the latest version (%s)", CurrentVersion)
	}

	assetName := fmt.Sprintf("mncode-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}
	asset, err := selectReleaseAsset(manifest, assetName)
	if err != nil {
		return err
	}
	return downloadAndReplace(asset)
}

func downloadAndReplace(asset cliReleaseAsset) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find current executable path: %w", err)
	}
	return downloadReleaseAsset(asset, execPath)
}
