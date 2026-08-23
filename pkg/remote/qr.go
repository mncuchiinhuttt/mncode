package remote

import (
	qrcode "github.com/skip2/go-qrcode"
)

// GenerateTerminalQRCode returns a compact terminal-friendly string representation of the QR code
func GenerateTerminalQRCode(url string) string {
	qr, err := qrcode.New(url, qrcode.Low)
	if err != nil {
		return ""
	}
	return qr.ToSmallString(false)
}
