package ui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	lastCopiedText  string
	copyToastMu     sync.Mutex
	activeCopyToast string
	toastClearTimer *time.Timer
	onCopyCallbacks []func()
	stopWatcher     chan struct{}
)

// SetCopyCallback sets the active prompt copy hook and returns an unregister function
func SetCopyCallback(fn func()) func() {
	copyToastMu.Lock()
	onCopyCallbacks = []func(){fn}
	copyToastMu.Unlock()

	return func() {
		copyToastMu.Lock()
		onCopyCallbacks = nil
		copyToastMu.Unlock()
	}
}

// RegisterCopyCallback registers a hook called whenever text is copied
func RegisterCopyCallback(fn func()) func() {
	return SetCopyCallback(fn)
}

// GetActiveCopyToast returns the current toast message if active
func GetActiveCopyToast() string {
	copyToastMu.Lock()
	defer copyToastMu.Unlock()
	return activeCopyToast
}

// ShowCopyToast sets active toast message in status bar and triggers UI refresh
func ShowCopyToast(charCount int) {
	if charCount <= 0 {
		return
	}
	copyToastMu.Lock()
	activeCopyToast = fmt.Sprintf("\033[1;32m[OK]\033[0m \033[38;5;218mCopied %d characters to clipboard\033[0m", charCount)
	if toastClearTimer != nil {
		toastClearTimer.Stop()
	}
	toastClearTimer = time.AfterFunc(3*time.Second, func() {
		copyToastMu.Lock()
		activeCopyToast = ""
		copyToastMu.Unlock()
		triggerCopyCallbacks()
	})
	copyToastMu.Unlock()

	triggerCopyCallbacks()
}

func triggerCopyCallbacks() {
	copyToastMu.Lock()
	cbs := append([]func(){}, onCopyCallbacks...)
	copyToastMu.Unlock()
	for _, fn := range cbs {
		fn()
	}
}

// CopyToClipboard copies text to the OS clipboard and emits OSC 52 sequence
func CopyToClipboard(text string) error {
	if text == "" {
		return nil
	}

	lastCopiedText = text

	// 1. Emit OSC 52 sequence
	b64 := base64.StdEncoding.EncodeToString([]byte(text))
	fmt.Printf("\033]52;c;%s\007", b64)

	// 2. Native OS clipboard command
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	case "windows":
		cmd = exec.Command("clip")
	}

	if cmd != nil {
		inPipe, err := cmd.StdinPipe()
		if err == nil {
			_ = cmd.Start()
			_, _ = inPipe.Write([]byte(text))
			_ = inPipe.Close()
			_ = cmd.Wait()
		}
	}

	ShowCopyToast(len([]rune(text)))
	return nil
}

// GetClipboardText reads current text from system clipboard
func GetClipboardText() string {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		if _, err := exec.LookPath("wl-paste"); err == nil {
			cmd = exec.Command("wl-paste")
		} else if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
		}
	default:
		return ""
	}

	if cmd == nil {
		return ""
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return out.String()
}

// StartClipboardWatcher monitors clipboard for user selections and displays copy count
func StartClipboardWatcher() func() {
	lastCopiedText = strings.TrimSpace(GetClipboardText())
	stopWatcher = make(chan struct{})

	go func() {
		ticker := time.NewTicker(350 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopWatcher:
				return
			case <-ticker.C:
				current := strings.TrimSpace(GetClipboardText())
				if current != "" && current != lastCopiedText {
					lastCopiedText = current
					charCount := len([]rune(current))
					ShowCopyToast(charCount)
				}
			}
		}
	}()

	return func() {
		if stopWatcher != nil {
			close(stopWatcher)
		}
	}
}
