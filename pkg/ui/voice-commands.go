package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mncode/pkg/agent"
	"mncode/pkg/voice"
)

func handleVoiceCommand(args string, s *agent.Session) {
	if s == nil {
		fmt.Println("\033[31m[Error] Active session required for /voice.\033[0m")
		return
	}

	seconds := 6
	parts := strings.Fields(strings.TrimSpace(args))
	if len(parts) > 0 {
		if sec, err := strconv.Atoi(parts[0]); err == nil && sec > 0 && sec <= 30 {
			seconds = sec
		}
	}

	apiKey := ""
	if s.Config != nil {
		apiKey = s.Config.APIKey
	}

	tmpAudio := filepath.Join(os.TempDir(), fmt.Sprintf("mncode-voice-%d.wav", time.Now().UnixNano()))
	defer func() { _ = os.Remove(tmpAudio) }()

	fmt.Println("\n\033[1;36m=== Voice-to-Code Input Mode ===\033[0m")
	fmt.Printf("Listening from microphone for %d seconds... Speak your task!\n", seconds)

	err := voice.RecordMicAudio(context.Background(), tmpAudio, seconds)
	if err != nil {
		fmt.Printf("\033[33m[Notice] Microphone record tool unavailable (%v).\033[0m\n", err)
		fmt.Print("Type your voice transcript or prompt manually:\n> ")
		text := strings.TrimSpace(readLineRaw())
		if text != "" {
			_ = s.ProcessUserInput(context.Background(), text)
		}
		return
	}

	fmt.Println("Transcribing speech with Whisper STT...")
	transcribed, err := voice.TranscribeAudio(context.Background(), apiKey, tmpAudio)
	if err != nil {
		fmt.Printf("\033[31m[STT Error] %v\033[0m\n", err)
		return
	}

	if transcribed == "" {
		fmt.Println("\033[33m[Notice] No speech detected in audio recording.\033[0m")
		return
	}

	fmt.Printf("\n\033[1;32m[Voice Captured]:\033[0m %s\n\n", transcribed)
	_ = s.ProcessUserInput(context.Background(), transcribed)
}
