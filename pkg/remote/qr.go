package remote

import (
	qrcode "github.com/skip2/go-qrcode"
)

// GenerateTerminalQRCode returns a terminal-friendly string representation of the QR code
func GenerateTerminalQRCode(url string) string {
	qr, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return ""
	}
	return qr.ToSmallString(false)
}
