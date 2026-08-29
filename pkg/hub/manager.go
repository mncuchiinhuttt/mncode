package hub

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Manager manages project-scoped long-running background services.
type Manager struct {
	mu        sync.RWMutex
	processes map[string]*SupervisedProcess
}

var defaultGlobalManager *Manager
var managerOnce sync.Once

// GlobalManager returns the project-scoped singleton Process Hub manager.
func GlobalManager() *Manager {
	managerOnce.Do(func() {
		defaultGlobalManager = &Manager{
			processes: make(map[string]*SupervisedProcess),
		}
	})
	return defaultGlobalManager
}

// Start launches a new background service and waits for configured readiness conditions.
func (m *Manager) Start(ctx context.Context, spec ServiceSpec) (*ServiceInfo, error) {
	m.mu.Lock()
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		m.mu.Unlock()
		return nil, fmt.Errorf("service name is required")
	}
	if spec.Command == "" {
		m.mu.Unlock()
		return nil, fmt.Errorf("service command is required")
	}

	if existing, exists := m.processes[name]; exists && existing.state == StateRunning {
		m.mu.Unlock()
		return nil, fmt.Errorf("service %q is already running (PID %d)", name, existing.cmd.Process.Pid)
	}

	p := newSupervisedProcess(spec)
	m.processes[name] = p
	m.mu.Unlock()

	if err := p.start(); err != nil {
		return nil, fmt.Errorf("start service %q: %w", name, err)
	}

	timeout := time.Duration(spec.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 1. Probe ReadyPort if specified
	if spec.ReadyPort > 0 {
		if err := WaitForPort(ctx, "127.0.0.1", spec.ReadyPort, timeout); err != nil {
			_ = p.Stop()
			return nil, fmt.Errorf("service %q TCP port %d readiness failed: %w", name, spec.ReadyPort, err)
		}
	}

	// 2. Probe ReadyLog regex if specified
	if spec.ReadyLog != "" {
		if err := WaitForLogRegex(ctx, p.linesChan, spec.ReadyLog, timeout); err != nil {
			_ = p.Stop()
			return nil, fmt.Errorf("service %q log regex readiness failed: %w", name, err)
		}
	}

	return &ServiceInfo{
		Name:      name,
		PID:       p.cmd.Process.Pid,
		Command:   spec.Command,
		State:     p.state,
		ReadyPort: spec.ReadyPort,
		StartTime: p.startTime,
	}, nil
}

// Logs returns recent log lines from the specified service.
func (m *Manager) Logs(name string, limit int, grep string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.processes[name]
	if !ok {
		return nil, fmt.Errorf("service %q not found", name)
	}
	return p.GetLogs(limit, grep), nil
}

// Send writes input text or commands to the service's stdin.
func (m *Manager) Send(name string, text string, enter bool) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.processes[name]
	if !ok {
		return fmt.Errorf("service %q not found", name)
	}
	return p.SendText(text, enter)
}

// Stop shuts down the named service process tree.
func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.processes[name]
	if !ok {
		return fmt.Errorf("service %q not found", name)
	}
	err := p.Stop()
	delete(m.processes, name)
	return err
}

// PS returns status information for all registered services.
func (m *Manager) PS() []ServiceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []ServiceInfo
	for name, p := range m.processes {
		pid := 0
		if p.cmd != nil && p.cmd.Process != nil {
			pid = p.cmd.Process.Pid
		}
		dur := 0.0
		if !p.startTime.IsZero() {
			dur = time.Since(p.startTime).Seconds()
		}
		list = append(list, ServiceInfo{
			Name:        name,
			PID:         pid,
			Command:     p.Spec.Command,
			State:       p.state,
			ReadyPort:   p.Spec.ReadyPort,
			StartTime:   p.startTime,
			DurationSec: dur,
		})
	}
	return list
}

// CloseAll shuts down every running service upon session exit.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.processes {
		_ = p.Stop()
	}
	m.processes = make(map[string]*SupervisedProcess)
}
