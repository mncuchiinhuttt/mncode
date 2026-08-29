package voice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTranscribeAudioNonExistentFile(t *testing.T) {
	_, err := TranscribeAudio(context.Background(), "mock-key", "/nonexistent/audio.wav")
	if err == nil {
		t.Fatal("expected error when transcribing nonexistent file, got nil")
	}
}

func TestRecordMicAudioTimeout(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-voice-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	outPath := filepath.Join(tempDir, "test.wav")
	// Will attempt ffmpeg/rec and gracefully fail with structured error if tools are not present
	_ = RecordMicAudio(context.Background(), outPath, 1)
}
