package ui

import (
	"math/rand"
	"mncode/pkg/agent"
	"sync"
	"time"
)

var (
	brainrotIdleLines = []string{
		"Bro is locked in thoughts or did you get fanum taxed by the compiler? [GIGA]",
		"Aura -1000 for staying idle... Type something king [KING]",
		"Are we cooking or just staring at the terminal like an NPC? [COOK]",
		"Bro's Mewing streak is longer than this pause [MEW]",
		"Skibidi code won't write itself fr fr no cap [NO CAP]",
		"Level 100 Sigma Dev detected: Waiting for the ultimate prompt... [MAX]",
		"Brainrot energy at 99%: Give me tasks to mog this codebase [SIGMA]",
		"Bro is generating infinite aura before hitting Enter [GIGA][ACTION]",
		"Did you get distracted by Subway Surfers gameplay below? 🛹",
		"Max rizz compiler standing by: Feed me code to cook! 🍲",
	}

	idleMu           sync.Mutex
	activeIdleRizz   string
	lastActivityTime = time.Now()
	idleRizzIndex    = 0
)

// GetActiveIdleRizz returns the current idle rizz line if active
func GetActiveIdleRizz() string {
	idleMu.Lock()
	defer idleMu.Unlock()
	return activeIdleRizz
}

// ResetBrainrotActivity resets the activity timer and clears idle rizz line
func ResetBrainrotActivity(onNudge func()) {
	idleMu.Lock()
	lastActivityTime = time.Now()
	wasActive := activeIdleRizz != ""
	activeIdleRizz = ""
	idleMu.Unlock()

	if wasActive && onNudge != nil {
		onNudge()
	}
}

// StartBrainrotIdleWatcher monitors for inactivity in brainrot mode
func StartBrainrotIdleWatcher(s *agent.Session, onNudge func()) func() {
	stopCh := make(chan struct{})

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				if s.Config.GetSetting("brainrot_mode", "false") != "true" {
					continue
				}

				idleMu.Lock()
				idleDuration := time.Since(lastActivityTime)
				needsUpdate := false

				if idleDuration >= 50*time.Second && activeIdleRizz == "" {
					idleRizzIndex = rand.Intn(len(brainrotIdleLines))
					activeIdleRizz = brainrotIdleLines[idleRizzIndex]
					needsUpdate = true
				}
				idleMu.Unlock()

				if needsUpdate && onNudge != nil {
					onNudge()
				}
			}
		}
	}()

	return func() {
		close(stopCh)
		idleMu.Lock()
		activeIdleRizz = ""
		idleMu.Unlock()
	}
}
