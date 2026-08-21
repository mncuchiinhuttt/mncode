package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// TrySaveClipboardImage extracts an image from system clipboard and saves to workspace cache
func TrySaveClipboardImage(workspaceDir string) (string, bool) {
	imagesDir := filepath.Join(workspaceDir, ".mncode", "images")
	_ = os.MkdirAll(imagesDir, 0755)

	targetFile := filepath.Join(imagesDir, fmt.Sprintf("paste-%s.png", time.Now().Format("20060102-150405")))

	switch runtime.GOOS {
	case "darwin":
		// Try pngpaste first if installed
		if _, err := exec.LookPath("pngpaste"); err == nil {
			cmd := exec.Command("pngpaste", targetFile)
			if err := cmd.Run(); err == nil {
				if fi, statErr := os.Stat(targetFile); statErr == nil && fi.Size() > 0 {
					return targetFile, true
				}
			}
		}
		// Fallback to osascript png export
		script := fmt.Sprintf(`
			try
				set pngData to the clipboard as «class PNGf»
				set outFile to open for access POSIX file "%s" with write permission
				set eof outFile to 0
				write pngData to outFile
				close access outFile
				return "ok"
			on error
				return "err"
			end try
		`, targetFile)
		cmd := exec.Command("osascript", "-e", script)
		out, err := cmd.Output()
		if err == nil && string(out) != "" {
			if fi, statErr := os.Stat(targetFile); statErr == nil && fi.Size() > 100 {
				return targetFile, true
			}
		}

	case "linux":
		if _, err := exec.LookPath("wl-paste"); err == nil {
			cmd := exec.Command("wl-paste", "--type", "image/png")
			data, err := cmd.Output()
			if err == nil && len(data) > 0 {
				_ = os.WriteFile(targetFile, data, 0644)
				return targetFile, true
			}
		}
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o")
			data, err := cmd.Output()
			if err == nil && len(data) > 0 {
				_ = os.WriteFile(targetFile, data, 0644)
				return targetFile, true
			}
		}

	case "windows":
		psCmd := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms;
			$img = [System.Windows.Forms.Clipboard]::GetImage();
			if ($img -ne $null) {
				$img.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png);
				Write-Output "ok"
			}
		`, targetFile)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
		if err := cmd.Run(); err == nil {
			if fi, statErr := os.Stat(targetFile); statErr == nil && fi.Size() > 0 {
				return targetFile, true
			}
		}
	}

	return "", false
}

// IsImageFile checks if a filepath points to an image
func IsImageFile(path string) bool {
	ext := stringsToLower(filepath.Ext(path))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp" || ext == ".gif" || ext == ".svg"
}

func stringsToLower(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b = append(b, c)
	}
	return string(b)
}
