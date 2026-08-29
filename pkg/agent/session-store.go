package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mncode/pkg/persistence"
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

func openCanonicalStore() (*persistence.Store, error) {
	return persistence.Open(context.Background(), persistence.StoreConfig{Profile: "default"})
}

func canonicalMessages(sessionID string, history []provider.Message, at time.Time) []persistence.MessageRecord {
	out := make([]persistence.MessageRecord, 0, len(history))
	for i, message := range history {
		payload, _ := json.Marshal(message)
		msgID := fmt.Sprintf("%s:msg:%d", sessionID, i)
		out = append(out, persistence.MessageRecord{
			ID: msgID, Sequence: i, Role: string(message.Role),
			Content: message.Content, Thinking: message.Thinking, Payload: payload, CreatedAt: at,
		})
	}
	return out
}

func savedSessionFromCanonical(record persistence.SessionRecord) *SavedSession {
	saved := &SavedSession{
		ID: record.ID, Title: record.Title, WorkspaceDir: record.WorkspaceDir,
		CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
		Model: record.Model, Turns: record.Turns,
	}
	for _, stored := range record.Messages {
		var message provider.Message
		if len(stored.Payload) > 0 {
			_ = json.Unmarshal(stored.Payload, &message)
		}
		if message.Role == "" {
			message.Role = provider.Role(stored.Role)
			message.Content = stored.Content
			message.Thinking = stored.Thinking
		}
		saved.Messages = append(saved.Messages, message)
	}

	return saved
}
func importLegacySessionBestEffort(path string, data []byte) {
	canonical, err := openCanonicalStore()
	if err != nil {
		return
	}
	defer canonical.Close()
	backupPath := path + ".backup"
	if _, statErr := os.Stat(backupPath); os.IsNotExist(statErr) {
		if _, backupErr := persistence.Backup(path, backupPath); backupErr != nil {
			return
		}
	}
	_, _, _ = persistence.ImportLegacySessionJSON(context.Background(), canonical, data, backupPath)
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

	now := time.Now().UTC()
	model, providerName := "", ""
	if s.Config != nil {
		model, providerName = s.Config.Model, string(s.Config.Provider)
	}
	saved := SavedSession{
		ID: s.ID, Title: title, WorkspaceDir: s.WorkspaceDir,
		CreatedAt: now, UpdatedAt: now, Model: model,
		Turns: turns, Messages: s.History,
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
	// Keep writing the legacy file as an explicit compatibility export until
	// all callers migrate; SQLite failure safely falls back to this JSON path.
	var canonicalErr error
	if canonical, err := openCanonicalStore(); err == nil {
		canonicalErr = canonical.SaveSession(context.Background(), persistence.SessionRecord{
			ID: s.ID, Title: title, WorkspaceDir: s.WorkspaceDir, ChatID: s.ID,
			Model: model, Provider: providerName, Turns: turns,
			CreatedAt: now, UpdatedAt: now, Messages: canonicalMessages(s.ID, s.History, now),
		})
		if closeErr := canonical.Close(); canonicalErr == nil {
			canonicalErr = closeErr
		}
	} else if !errors.Is(err, persistence.ErrSQLiteUnavailable) {
		canonicalErr = err
	}
	if err := writePrivateFile(filePath, data); err != nil {
		return err
	}
	if canonicalErr != nil {
		return canonicalErr
	}
	return nil
}

func ListSavedSessions() ([]*SavedSession, error) {
	// Read canonical state first, then merge legacy files so an interrupted
	// import never hides recoverable source data. Canonical rows win by ID.
	var list []*SavedSession
	seen := make(map[string]bool)
	if canonical, err := openCanonicalStore(); err == nil {
		records, listErr := canonical.ListSessions(context.Background(), persistence.SearchFilter{})
		closeErr := canonical.Close()
		if listErr != nil {
			return nil, listErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		for _, record := range records {
			saved := savedSessionFromCanonical(record)
			if len(saved.Messages) > 0 {
				list = append(list, saved)
				seen[saved.ID] = true
			}
		}
	} else if !errors.Is(err, persistence.ErrSQLiteUnavailable) {
		return nil, err
	}

	dir := GetSessionsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			continue
		}
		var saved SavedSession
		if jsonErr := json.Unmarshal(data, &saved); jsonErr == nil && len(saved.Messages) > 0 && !seen[saved.ID] {
			importLegacySessionBestEffort(filepath.Join(dir, entry.Name()), data)
			list = append(list, &saved)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].UpdatedAt.After(list[j].UpdatedAt) })
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
	if strings.TrimSpace(id) == "" || strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
		return nil, fmt.Errorf("invalid session id")
	}
	if canonical, err := openCanonicalStore(); err == nil {
		record, getErr := canonical.GetSession(context.Background(), id)
		closeErr := canonical.Close()
		if getErr == nil {
			if closeErr != nil {
				return nil, closeErr
			}
			return savedSessionFromCanonical(record), nil
		}
		if !errors.Is(getErr, persistence.ErrNotFound) {
			return nil, getErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	} else if !errors.Is(err, persistence.ErrSQLiteUnavailable) {
		return nil, err
	}
	dir := GetSessionsDir()
	fileName, err := safeSessionFilename(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return nil, err
	}
	var saved SavedSession
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, err
	}
	importLegacySessionBestEffort(filepath.Join(dir, fileName), data)
	return &saved, nil
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
	if err := replaceExistingFile(tmpPath, path); err != nil {
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
