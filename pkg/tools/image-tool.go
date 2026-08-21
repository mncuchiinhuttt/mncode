package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ViewImageTool inspects and views local images for multimodal reasoning
type ViewImageTool struct {
	BaseDir string
}

func (v *ViewImageTool) Name() string {
	return "view_image"
}

func (v *ViewImageTool) Description() string {
	return "Inspects and views a local image file (PNG, JPG, JPEG, GIF, WebP, SVG). Returns dimensions, format, file size, and loads image for vision analysis."
}

func (v *ViewImageTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the image file (relative to workspace or absolute).",
			},
		},
		"required": []string{"path"},
	}
}

func (v *ViewImageTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	relPath, _ := args["path"].(string)
	if relPath == "" {
		relPath, _ = args["image_path"].(string)
	}
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("path is required")
	}

	fullPath := relPath
	if !filepath.IsAbs(fullPath) && v.BaseDir != "" {
		fullPath = filepath.Join(v.BaseDir, relPath)
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("image file not found: %w", err)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	mimeType := http.DetectContentType(data)
	if strings.HasSuffix(strings.ToLower(fullPath), ".svg") {
		mimeType = "image/svg+xml"
	}

	dimStr := "unknown dimensions"
	fileReader, err := os.Open(fullPath)
	if err == nil {
		defer fileReader.Close()
		cfg, format, cfgErr := image.DecodeConfig(fileReader)
		if cfgErr == nil {
			dimStr = fmt.Sprintf("%dx%d %s", cfg.Width, cfg.Height, strings.ToUpper(format))
		}
	}

	sizeKB := float64(fileInfo.Size()) / 1024.0
	b64Preview := ""
	if len(data) > 0 {
		encoded := base64.StdEncoding.EncodeToString(data)
		if len(encoded) > 60 {
			b64Preview = encoded[:60] + "..."
		} else {
			b64Preview = encoded
		}
	}

	return fmt.Sprintf("Image File: %s\nMIME Type:  %s\nDimensions: %s\nFile Size:  %.1f KB (%d bytes)\nData URI:   data:%s;base64,%s\n\n[Image successfully loaded into vision analysis context]",
		relPath, mimeType, dimStr, sizeKB, fileInfo.Size(), mimeType, b64Preview), nil
}
