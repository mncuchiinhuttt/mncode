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
	WorkspaceDir   string
	Config         *Config
	ConfigPath     string
	IsWorkspaceLvl bool
	IsTrusted      bool
	Clients        map[string]*Client
	mu             sync.RWMutex
}

func getTrustedWorkspacesPath() string {
	return filepath.Join(os.Getenv("HOME"), ".mncode", "trusted_workspaces.json")
}

func isWorkspaceTrusted(workspaceDir string) bool {
	if workspaceDir == "" {
		return true
	}
	cleanWs, err := filepath.Abs(workspaceDir)
	if err != nil {
		cleanWs = workspaceDir
	}

	data, err := os.ReadFile(getTrustedWorkspacesPath())
	if err != nil {
		return false
	}
	var trusted []string
	if err := json.Unmarshal(data, &trusted); err != nil {
		return false
	}
	for _, p := range trusted {
		if absP, err := filepath.Abs(p); err == nil && absP == cleanWs {
			return true
		}
	}
	return false
}

// TrustWorkspace records a workspace directory as trusted for MCP execution
func TrustWorkspace(workspaceDir string) error {
	if workspaceDir == "" {
		return nil
	}
	cleanWs, err := filepath.Abs(workspaceDir)
	if err != nil {
		cleanWs = workspaceDir
	}

	p := getTrustedWorkspacesPath()
	var trusted []string
	if data, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(data, &trusted)
	}

	for _, t := range trusted {
		if absT, err := filepath.Abs(t); err == nil && absT == cleanWs {
			return nil
		}
	}

	trusted = append(trusted, cleanWs)
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	data, err := json.MarshalIndent(trusted, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
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

	workspacePaths := []string{
		filepath.Join(m.WorkspaceDir, "mcp.json"),
		filepath.Join(m.WorkspaceDir, ".claude", "mcp.json"),
	}
	globalPath := filepath.Join(os.Getenv("HOME"), ".mncode", "mcp.json")

	var foundPath string
	var cfg Config
	var isWorkspace bool

	if m.WorkspaceDir != "" {
		for _, p := range workspacePaths {
			if data, err := os.ReadFile(p); err == nil {
				if err := json.Unmarshal(data, &cfg); err == nil {
					foundPath = p
					isWorkspace = true
					break
				}
			}
		}
	}

	if foundPath == "" {
		if data, err := os.ReadFile(globalPath); err == nil {
			if err := json.Unmarshal(data, &cfg); err == nil {
				foundPath = globalPath
				isWorkspace = false
			}
		}
	}

	if foundPath == "" {
		foundPath = globalPath
		cfg = Config{MCPServers: make(map[string]ServerConfig)}
		isWorkspace = false
	}

	m.ConfigPath = foundPath
	m.Config = &cfg
	m.IsWorkspaceLvl = isWorkspace
	if isWorkspace {
		m.IsTrusted = isWorkspaceTrusted(m.WorkspaceDir)
	} else {
		m.IsTrusted = true
	}
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

	if err := os.WriteFile(m.ConfigPath, data, 0600); err != nil {
		return err
	}
	return os.Chmod(m.ConfigPath, 0600)
}

// UpsertServer saves a server definition without starting an external process.
// Hosts can call StartAll when a workspace session is ready.
func (m *Manager) UpsertServer(name string, cfg ServerConfig) error {
	if name == "" || cfg.Command == "" {
		return fmt.Errorf("MCP server name and command are required")
	}
	m.mu.Lock()
	if m.Config == nil {
		m.Config = &Config{MCPServers: make(map[string]ServerConfig)}
	}
	if m.Config.MCPServers == nil {
		m.Config.MCPServers = make(map[string]ServerConfig)
	}
	if client := m.Clients[name]; client != nil {
		_ = client.Close()
		delete(m.Clients, name)
	}
	m.Config.MCPServers[name] = cfg
	m.mu.Unlock()
	return m.SaveConfig()
}

func (m *Manager) GetServerConfig(name string) (ServerConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Config == nil || m.Config.MCPServers == nil {
		return ServerConfig{}, false
	}
	cfg, ok := m.Config.MCPServers[name]
	return cfg, ok
}

func (m *Manager) IsConnected(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.Clients[name]
	return ok && client != nil
}

func (m *Manager) StartAll(ctx context.Context) {
	m.mu.RLock()
	isWs := m.IsWorkspaceLvl
	isTrusted := m.IsTrusted
	servers := make(map[string]ServerConfig, len(m.Config.MCPServers))
	for name, cfg := range m.Config.MCPServers {
		servers[name] = cfg
	}
	m.mu.RUnlock()

	// Security Policy: Do not automatically spawn untrusted workspace-level MCP servers
	if isWs && !isTrusted {
		return
	}

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
