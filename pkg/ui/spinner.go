package ui

import (
	"fmt"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner provides animated terminal progress indicator with optional stopwatch timer
type Spinner struct {
	mu        sync.Mutex
	message   string
	active    bool
	startTime time.Time
	showTimer bool
	stopCh    chan struct{}
}

func NewSpinner(msg string) *Spinner {
	return &Spinner{
		message:   msg,
		stopCh:    make(chan struct{}),
		showTimer: true,
	}
}

func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	if s.startTime.IsZero() {
		s.startTime = time.Now()
	}
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	go func() {
		idx := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				fmt.Print("\r\033[K")
				return
			case <-ticker.C:
				s.mu.Lock()
				frame := spinnerFrames[idx%len(spinnerFrames)]
				msg := s.message
				st := s.startTime
				hasTimer := s.showTimer
				s.mu.Unlock()

				timerStr := ""
				if hasTimer && !st.IsZero() {
					elapsed := time.Since(st)
					mins := int(elapsed.Minutes())
					secs := int(elapsed.Seconds()) % 60
					timerStr = fmt.Sprintf("\033[38;5;218m[%02d:%02d]\033[0m ", mins, secs)
				}

				fmt.Printf("\r\033[36m%s\033[0m %s%s", frame, timerStr, msg)
				idx++
			}
		}
	}()
}

func (s *Spinner) ResetTimer() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startTime = time.Now()
}

func (s *Spinner) UpdateMessage(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = msg
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}
	s.active = false
	close(s.stopCh)
	time.Sleep(20 * time.Millisecond)
}
