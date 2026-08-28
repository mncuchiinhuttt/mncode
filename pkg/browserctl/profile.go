// Profile helpers: locate the user's real Chrome profile, selectively import
// safe browsing data (cookies/bookmarks/history — never saved passwords)
// into mncode's isolated controlled-browser profile, and clear that
// isolated profile's cache or all its data on request.
package browserctl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// chromeCandidatePaths returns plausible default Chrome profile directories
// per OS. Only the first one that exists is used.
func chromeCandidatePaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	switch runtime.GOOS {
	case "darwin":
		return []string{filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default")}
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return []string{filepath.Join(localAppData, "Google", "Chrome", "User Data", "Default")}
		}
		return nil
	default:
		return []string{filepath.Join(home, ".config", "google-chrome", "Default")}
	}
}

// FindChromeProfile returns the path to the user's default Chrome profile
// directory, or "" if none was found on this machine.
func FindChromeProfile() string {
	for _, candidate := range chromeCandidatePaths() {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

// importableFiles are the specific Chrome profile files mncode will copy
// into the isolated controlled-browser profile. Deliberately excludes
// "Login Data" (saved passwords) and "Web Data" (autofill/payment data) —
// import is for staying logged into sites via cookies, not for handling
// credential material.
var importableFiles = []string{
	"Cookies",
	"Cookies-journal",
	"Bookmarks",
	"History",
	"Favicons",
	filepath.Join("Network", "Cookies"),
	filepath.Join("Network", "Cookies-journal"),
}

// ImportChromeProfile copies cookies, bookmarks, and history from the
// user's real Chrome profile into destDir (mncode's isolated
// controlled-browser profile's "Default" directory). The controlled browser
// must not be running during import. Returns the number of files copied.
func ImportChromeProfile(destDir string) (int, error) {
	src := FindChromeProfile()
	if src == "" {
		return 0, fmt.Errorf("no default Chrome profile was found on this computer")
	}
	defaultDest := filepath.Join(destDir, "Default")
	if err := os.MkdirAll(defaultDest, 0o755); err != nil {
		return 0, fmt.Errorf("create profile directory: %w", err)
	}

	copied := 0
	for _, rel := range importableFiles {
		srcPath := filepath.Join(src, rel)
		if _, err := os.Stat(srcPath); err != nil {
			continue
		}
		dstPath := filepath.Join(defaultDest, rel)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			continue
		}
		if err := copyFile(srcPath, dstPath); err == nil {
			copied++
		}
	}
	if copied == 0 {
		return 0, fmt.Errorf("no importable Chrome profile data was found (cookies/bookmarks/history)")
	}
	return copied, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// cacheDirs are the cache-only subdirectories under a Chrome profile's
// "Default" directory. Removing these frees disk space and forces fresh
// asset fetches without touching cookies, logins, or history.
var cacheDirs = []string{
	"Cache",
	"Code Cache",
	"GPUCache",
	filepath.Join("Service Worker", "CacheStorage"),
	filepath.Join("Service Worker", "ScriptCache"),
	"DawnCache",
	"GrShaderCache",
	"ShaderCache",
}

// ClearBrowserCache removes only cache directories from the controlled
// browser's profile at dataDir, leaving cookies/history/bookmarks intact.
func ClearBrowserCache(dataDir string) error {
	defaultDir := filepath.Join(dataDir, "Default")
	var firstErr error
	removed := 0
	for _, rel := range cacheDirs {
		path := filepath.Join(defaultDir, rel)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		removed++
	}
	// Also clear the top-level GPU shader cache directory (lives outside Default/).
	topLevel := filepath.Join(dataDir, "ShaderCache")
	if _, err := os.Stat(topLevel); err == nil {
		_ = os.RemoveAll(topLevel)
	}
	if firstErr != nil {
		return firstErr
	}
	return nil
}

// ClearBrowserData wipes the entire controlled-browser profile directory
// (cookies, history, cache, everything), giving the next launch a
// completely fresh isolated profile. The user's real Chrome profile is
// never touched — this only ever operates on mncode's own profile dir.
func ClearBrowserData(dataDir string) error {
	if dataDir == "" {
		return fmt.Errorf("browser profile directory is required")
	}
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read browser profile directory: %w", err)
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dataDir, entry.Name())); err != nil {
			return fmt.Errorf("clear browser data: %w", err)
		}
	}
	return nil
}
