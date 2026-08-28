package browserctl

import "sync"

// manager holds the single process-wide controlled-browser Session. Both the
// agent's control_browser tool and the desktop settings screen (import
// profile / clear cache / clear data / close browser) share this instance,
// so "the browser the agent is driving" and "the browser shown in settings
// status" are always the same one.
var (
	managerMu sync.Mutex
	shared    *Session
	sharedOpt Options
)

// Shared returns the process-wide browser session, creating it with opts on
// first use. Subsequent calls ignore opts and return the existing session —
// use Reconfigure to apply changed settings (it closes and recreates).
func Shared(opts Options) *Session {
	managerMu.Lock()
	defer managerMu.Unlock()
	if shared == nil {
		sharedOpt = opts
		shared = NewSession(opts)
	}
	return shared
}

// HasShared reports whether a shared session has been created (regardless
// of whether the underlying browser process is currently running).
func HasShared() bool {
	managerMu.Lock()
	defer managerMu.Unlock()
	return shared != nil
}

// IsSharedRunning reports whether the shared browser process is alive.
func IsSharedRunning() bool {
	managerMu.Lock()
	s := shared
	managerMu.Unlock()
	if s == nil {
		return false
	}
	return s.IsRunning()
}

// CloseShared shuts down the shared browser session, if any, and clears the
// singleton so the next Shared() call starts a fresh process.
func CloseShared() error {
	managerMu.Lock()
	s := shared
	shared = nil
	managerMu.Unlock()
	if s == nil {
		return nil
	}
	return s.Close()
}

// SharedUserDataDir returns the profile directory the shared session was (or
// would be) configured with, falling back to DefaultUserDataDir.
func SharedUserDataDir() string {
	managerMu.Lock()
	defer managerMu.Unlock()
	if sharedOpt.UserDataDir != "" {
		return sharedOpt.UserDataDir
	}
	return DefaultUserDataDir()
}
