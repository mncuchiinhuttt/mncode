package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TranscribeAudio sends an audio file (wav/mp3/m4a/webm) to the Whisper STT endpoint.
func TranscribeAudio(ctx context.Context, apiKey, audioFilePath string) (string, error) {
	file, err := os.Open(audioFilePath)
	if err != nil {
		return "", fmt.Errorf("open audio file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(audioFilePath))
	if err != nil {
		return "", fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("copy audio payload: %w", err)
	}

	_ = writer.WriteField("model", "whisper-1")
	_ = writer.WriteField("language", "vi") // Default Vietnamese / multilingual auto-detect
	_ = writer.Close()

	endpoint := "https://api.openai.com/v1/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", fmt.Errorf("create whisper request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read whisper response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper returned HTTP %d: %s", resp.StatusCode, string(respData))
	}

	type whisperResp struct {
		Text  string `json:"text"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	var res whisperResp
	if err := json.Unmarshal(respData, &res); err != nil {
		return "", fmt.Errorf("parse whisper json: %w", err)
	}
	if res.Error != nil {
		return "", fmt.Errorf("whisper error: %s", res.Error.Message)
	}

	return strings.TrimSpace(res.Text), nil
}

// RecordMicAudio attempts recording audio using system ffmpeg, sox, or native tools.
func RecordMicAudio(ctx context.Context, outputPath string, seconds int) error {
	if seconds <= 0 {
		seconds = 5
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(seconds+2)*time.Second)
	defer cancel()

	// Try ffmpeg / sox
	cmd := exec.CommandContext(timeoutCtx, "ffmpeg", "-y", "-f", "avfoundation", "-i", ":0", "-t", fmt.Sprintf("%d", seconds), outputPath)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fallback to rec / sox
	cmd2 := exec.CommandContext(timeoutCtx, "rec", "-q", outputPath, "trim", "0", fmt.Sprintf("%d", seconds))
	if err := cmd2.Run(); err == nil {
		return nil
	}

	return fmt.Errorf("no supported audio recording tool (ffmpeg or sox) found on PATH")
}
