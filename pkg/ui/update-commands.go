package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/config"
	"runtime"
	"time"
)

var (
	cachedUpdateNotice string
)

// StartBackgroundVersionCheck checks for updates in background without blocking CLI startup
func StartBackgroundVersionCheck() {
	go func() {
		defer func() { _ = recover() }()
		time.Sleep(1 * time.Second)
		latest, hasUpdate, err := config.CheckLatestVersion()
		if err == nil && hasUpdate {
			cachedUpdateNotice = fmt.Sprintf("💡 Update available: %s ➔ %s. Type '/update' to upgrade!",
				config.CurrentVersion, BoldGreen(latest))
		}
	}()
}

// GetCachedUpdateNotice returns background update toast if available
func GetCachedUpdateNotice() string {
	return cachedUpdateNotice
}

// HandleUpdateCommand executes in-app self-update
func HandleUpdateCommand(parts []string, s *agent.Session) {
	fmt.Printf("\n%s Checking for updates on GitHub...\n", BoldCyan("[Update]"))
	latestTag, hasUpdate, err := config.CheckLatestVersion()
	if err != nil {
		fmt.Printf("%s Could not check updates: %v\n\n", BoldRed("Error:"), err)
		return
	}

	if !hasUpdate {
		fmt.Printf("%s You are already on the latest version (%s)!\n\n", BoldGreen("✓"), Bold(config.CurrentVersion))
		return
	}

	fmt.Printf("%s New version found: %s (Current: %s)\n", BoldGreen("✓"), BoldGreen(latestTag), config.CurrentVersion)
	fmt.Printf("Downloading and installing update for %s/%s...\n", runtime.GOOS, runtime.GOARCH)

	err = config.PerformSelfUpdate(latestTag)
	if err != nil {
		fmt.Printf("\n%s Update failed: %v\n", BoldRed("Error:"), err)
		fmt.Printf("You can run the install script directly:\n  curl -fsSL https://raw.githubusercontent.com/%s/main/install.sh | bash\n\n", config.GithubRepo)
		return
	}

	cachedUpdateNotice = ""
	fmt.Printf("\n%s Successfully updated mncode to %s!\n", BoldGreen("✓ [Success]"), BoldGreen(latestTag))
	fmt.Println("Please restart mncode to apply the new version.")
}

// ShowVersionInfo prints current version and build information
func ShowVersionInfo() {
	fmt.Printf("\n%s %s (%s/%s)\n", BoldCyan("mncode"), Bold(config.CurrentVersion), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Repository: https://github.com/%s\n\n", config.GithubRepo)
}
