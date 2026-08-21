package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Manager struct {
	WorkspaceDir string
	Config       *Config
	ConfigPath   string
	Clients      map[string]*Client
	mu           sync.RWMutex
}

func NewManager(workspaceDir string) *Manager {
	m := &Manager{
		WorkspaceDir: workspaceDir,
		Clients:      make(map[string]*Client),
	}
	m.LoadConfig()
	return m
}

func (m *Manager) LoadConfig() {
	m.mu.Lock()
	defer m.mu.Unlock()

	paths := []string{
		filepath.Join(m.WorkspaceDir, "mcp.json"),
		filepath.Join(m.WorkspaceDir, ".claude", "mcp.json"),
		filepath.Join(os.Getenv("HOME"), ".mncode", "mcp.json"),
	}

	var foundPath string
	var cfg Config

	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			if err := json.Unmarshal(data, &cfg); err == nil {
				foundPath = p
				break
			}
		}
	}

	if foundPath == "" {
		foundPath = filepath.Join(os.Getenv("HOME"), ".mncode", "mcp.json")
		cfg = Config{MCPServers: make(map[string]ServerConfig)}
	}

	m.ConfigPath = foundPath
	m.Config = &cfg
}

func (m *Manager) SaveConfig() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dir := filepath.Dir(m.ConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.Config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.ConfigPath, data, 0644)
}

func (m *Manager) StartAll(ctx context.Context) {
	m.mu.RLock()
	servers := m.Config.MCPServers
	m.mu.RUnlock()

	var wg sync.WaitGroup
	for name, cfg := range servers {
		if cfg.Disabled {
			continue
		}
		wg.Add(1)
		go func(sName string, sCfg ServerConfig) {
			defer wg.Done()
			client, err := NewClient(sName, sCfg)
			if err == nil {
				m.mu.Lock()
				m.Clients[sName] = client
				m.mu.Unlock()
			}
		}(name, cfg)
	}
	wg.Wait()
}

func (m *Manager) AddServer(ctx context.Context, name string, cfg ServerConfig) error {
	client, err := NewClient(name, cfg)
	if err != nil {
		return fmt.Errorf("failed to start server '%s': %w", name, err)
	}

	m.mu.Lock()
	if m.Clients[name] != nil {
		_ = m.Clients[name].Close()
	}
	m.Clients[name] = client
	if m.Config.MCPServers == nil {
		m.Config.MCPServers = make(map[string]ServerConfig)
	}
	m.Config.MCPServers[name] = cfg
	m.mu.Unlock()

	return m.SaveConfig()
}

func (m *Manager) RemoveServer(name string) error {
	m.mu.Lock()
	if client, ok := m.Clients[name]; ok {
		_ = client.Close()
		delete(m.Clients, name)
	}
	if m.Config.MCPServers != nil {
		delete(m.Config.MCPServers, name)
	}
	m.mu.Unlock()

	return m.SaveConfig()
}

func (m *Manager) GetStatus(ctx context.Context) []ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []ServerStatus
	for name, cfg := range m.Config.MCPServers {
		stat := ServerStatus{
			Name:    name,
			Command: cfg.Command,
			Args:    cfg.Args,
		}

		client, active := m.Clients[name]
		if active && client != nil {
			stat.Connected = true
			tools, err := client.ListTools(ctx)
			if err != nil {
				stat.Error = err.Error()
			} else {
				stat.Tools = tools
			}
		} else if cfg.Disabled {
			stat.Error = "disabled"
		} else {
			stat.Error = "not connected"
		}
		list = append(list, stat)
	}
	return list
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, client := range m.Clients {
		_ = client.Close()
	}
	m.Clients = make(map[string]*Client)
}
