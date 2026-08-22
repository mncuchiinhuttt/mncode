package ui

import (
	"context"
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

// StartExecutionSteerWatcher listens for user keystrokes, Ctrl+C cancellation, and live steering during agent execution
func StartExecutionSteerWatcher(s *agent.Session, cancel context.CancelFunc) func() {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return func() {}
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return func() {}
	}

	stopChan := make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	var steerBuf []rune

	cleanup := func() {
		once.Do(func() {
			close(stopChan)
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
		})
	}

	go func() {
		buf := make([]byte, 128)
		for {
			select {
			case <-stopChan:
				return
			default:
			}

			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}

			select {
			case <-stopChan:
				return
			default:
			}

			mu.Lock()
			for i := 0; i < n; i++ {
				b := buf[i]

				// 1. Ctrl+C (ASCII 3) or Esc (ASCII 27 when single) -> Cancel Execution
				if b == 3 || (b == 27 && n == 1) {
					if cancel != nil {
						cancel()
					}
					fmt.Print("\r\n\033[1;31m[Cancelled by User]\033[0m\r\n")
					steerBuf = nil
					mu.Unlock()
					return
				}

				// 2. Enter -> Submit Steer or Queued Message
				if b == '\r' || b == '\n' {
					typed := strings.TrimSpace(string(steerBuf))
					steerBuf = nil
					if typed != "" {
						if strings.HasPrefix(typed, "/queue ") {
							qMsg := strings.TrimPrefix(typed, "/queue ")
							s.EnqueueMessage(qMsg)
							fmt.Printf("\r\n%s %s\r\n", BoldCyan("📥 [Message Queued for Next Turn]:"), Bold(qMsg))
						} else {
							s.EnqueueSteer(typed)
							fmt.Printf("\r\n%s %s\r\n", BoldPastelPink("⚡ [Steer Directive Injected (High Priority)]:"), Bold(typed))
						}
					}
					continue
				}

				// 3. Backspace
				if b == 127 || b == 8 {
					if len(steerBuf) > 0 {
						steerBuf = steerBuf[:len(steerBuf)-1]
						fmt.Print("\b \b")
					}
					continue
				}

				// 4. Printable characters (ASCII 32 to 126 and UTF-8)
				if b >= 32 && b <= 126 {
					steerBuf = append(steerBuf, rune(b))
					fmt.Print(string(b))
				}
			}
			mu.Unlock()
		}
	}()

	return cleanup
}
