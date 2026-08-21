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
	_ = os.MkdirAll(dir, 0755)
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

	filePath := filepath.Join(dir, fmt.Sprintf("%s.json", s.ID))
	return os.WriteFile(filePath, data, 0644)
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
	if !strings.HasSuffix(id, ".json") {
		id = id + ".json"
	}
	data, err := os.ReadFile(filepath.Join(dir, id))
	if err != nil {
		return nil, err
	}
	var s SavedSession
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Session) Restore(saved *SavedSession) {
	s.ID = saved.ID
	s.History = saved.Messages
	if saved.Model != "" && s.Config != nil {
		s.Config.Model = saved.Model
	}
}
