package agent

import (
	"encoding/json"
	"fmt"
	"mncode/pkg/provider"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SavedSession struct {
	ID           string             `json:"id"`
	Title        string             `json:"title"`
	WorkspaceDir string             `json:"workspaceDir"`
	CreatedAt    time.Time          `json:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt"`
	Model        string             `json:"model"`
	Turns        int                `json:"turns"`
	Messages     []provider.Message `json:"messages"`
}

func GetSessionsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".mncode", "sessions")
	_ = os.MkdirAll(dir, 0o700)
	_ = os.Chmod(dir, 0o700)
	return dir
}

func (s *Session) Save() error {
	if len(s.History) == 0 {
		return nil
	}
	dir := GetSessionsDir()
	if s.ID == "" || s.ID == "mncode-main" {
		s.ID = fmt.Sprintf("session-%s", time.Now().Format("20060102-150405"))
	}

	title := "New Session"
	turns := 0
	for _, m := range s.History {
		if m.Role == provider.RoleUser {
			turns++
			if title == "New Session" && m.Content != "" {
				title = strings.TrimSpace(m.Content)
				if len([]rune(title)) > 45 {
					title = string([]rune(title)[:42]) + "…"
				}
			}
		}
	}

	saved := SavedSession{
		ID:           s.ID,
		Title:        title,
		WorkspaceDir: s.WorkspaceDir,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Model:        s.Config.Model,
		Turns:        turns,
		Messages:     s.History,
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}

	fileName, err := safeSessionFilename(s.ID)
	if err != nil {
		return err
	}
	filePath := filepath.Join(dir, fileName)
	return writePrivateFile(filePath, data)
}

func ListSavedSessions() ([]*SavedSession, error) {
	dir := GetSessionsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var list []*SavedSession
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var s SavedSession
		if err := json.Unmarshal(data, &s); err == nil && len(s.Messages) > 0 {
			list = append(list, &s)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})

	return list, nil
}

func GetLatestSavedSession() (*SavedSession, error) {
	list, err := ListSavedSessions()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("no previous sessions found")
	}
	return list[0], nil
}

func LoadSavedSession(id string) (*SavedSession, error) {
	dir := GetSessionsDir()
	fileName, err := safeSessionFilename(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return nil, err
	}
	var s SavedSession
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func safeSessionFilename(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("session id is required")
	}
	if strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
		return "", fmt.Errorf("invalid session id")
	}
	if !strings.HasSuffix(id, ".json") {
		id += ".json"
	}
	if filepath.Base(id) != id || len(id) > 255 {
		return "", fmt.Errorf("invalid session id")
	}
	return id, nil
}

func writePrivateFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".session.json.tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func (s *Session) Restore(saved *SavedSession) {
	s.ID = saved.ID
	s.History = saved.Messages
	if saved.Model != "" && s.Config != nil {
		s.Config.Model = saved.Model
	}
}
