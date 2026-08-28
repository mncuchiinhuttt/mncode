package agent

import (
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mncode/pkg/provider"
	"mncode/pkg/tools"
)

var (
	imageTagRegex = regexp.MustCompile(`\[Image:\s*([^\]]+)\]`)
	atImageRegex  = regexp.MustCompile(`@image:([^\s]+)`)
)

// ExtractImagesFromInput finds image references in user prompt, loads bytes, and converts to base64
func ExtractImagesFromInput(workspaceDir, userInput string) (string, []provider.ImageData) {
	var images []provider.ImageData
	cleanInput := userInput

	// 1. Process [Image: path] tags
	matches := imageTagRegex.FindAllStringSubmatch(userInput, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		imgRelPath := strings.TrimSpace(m[1])
		imgData, ok := loadImageData(workspaceDir, imgRelPath)
		if ok {
			images = append(images, imgData)
		}
	}

	// 2. Process @image:path tags
	atMatches := atImageRegex.FindAllStringSubmatch(userInput, -1)
	for _, m := range atMatches {
		if len(m) < 2 {
			continue
		}
		imgRelPath := strings.TrimSpace(m[1])
		imgData, ok := loadImageData(workspaceDir, imgRelPath)
		if ok {
			images = append(images, imgData)
		}
	}

	return cleanInput, images
}

const maxImageAttachmentBytes = 10 << 20

func loadImageData(workspaceDir, relPath string) (provider.ImageData, bool) {
	fullPath := relPath
	if workspaceDir != "" {
		resolved, err := tools.ResolveWorkspacePath(workspaceDir, relPath, false)
		if err != nil {
			return provider.ImageData{}, false
		}
		fullPath = resolved
	} else if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Clean(fullPath)
	}

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() || info.Size() > maxImageAttachmentBytes {
		return provider.ImageData{}, false
	}
	data, err := os.ReadFile(fullPath)
	if err != nil || len(data) == 0 || len(data) > maxImageAttachmentBytes {
		return provider.ImageData{}, false
	}

	mimeType := http.DetectContentType(data)
	if strings.HasSuffix(strings.ToLower(fullPath), ".svg") {
		mimeType = "image/svg+xml"
	}

	return provider.ImageData{
		MediaType: mimeType,
		Data:      base64.StdEncoding.EncodeToString(data),
		FilePath:  relPath,
	}, true
}
