package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxArtifacts = 1000
	artifactDirPerm     = 0700
	artifactFilePerm    = 0600
)

// Store manages local virtual artifacts with strict privacy and file permissions.
type Store struct {
	mu      sync.RWMutex
	dir     string
	maxDocs int
}

var defaultGlobalStore *Store
var storeOnce sync.Once

// GlobalStore returns the shared project-wide artifact store.
func GlobalStore() *Store {
	storeOnce.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir := filepath.Join(home, ".mncode", "artifacts")
		_ = os.MkdirAll(dir, artifactDirPerm)
		defaultGlobalStore = &Store{dir: dir, maxDocs: defaultMaxArtifacts}
	})
	return defaultGlobalStore
}

// NewStore creates an artifact store backed by the specified directory.
func NewStore(dir string) (*Store, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".mncode", "artifacts")
	}
	if err := os.MkdirAll(dir, artifactDirPerm); err != nil {
		return nil, fmt.Errorf("create artifact directory: %w", err)
	}
	return &Store{dir: dir, maxDocs: defaultMaxArtifacts}, nil
}

// Save stores content under an 8-hex hash ID after automated credential scrubbing.
func (s *Store) Save(content string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	scrubbed := ScrubSecrets(content)
	hash := sha256.Sum256([]byte(scrubbed + fmt.Sprintf("-%d", time.Now().UnixNano())))
	id := hex.EncodeToString(hash[:])[:8]

	targetPath := filepath.Join(s.dir, fmt.Sprintf("%s.txt", id))
	tmpPath := fmt.Sprintf("%s.tmp-%d", targetPath, time.Now().UnixNano())

	if err := os.WriteFile(tmpPath, []byte(scrubbed), artifactFilePerm); err != nil {
		return "", fmt.Errorf("write temp artifact: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("commit artifact: %w", err)
	}

	s.evictLRULocked()
	return id, nil
}

// Get reads the artifact content by ID.
func (s *Store) Get(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id = strings.TrimPrefix(strings.TrimSpace(id), "artifact://")
	if !safeArtifactID(id) {
		return "", fmt.Errorf("invalid artifact id")
	}
	targetPath := filepath.Join(s.dir, fmt.Sprintf("%s.txt", id))
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return "", fmt.Errorf("artifact %q not found: %w", id, err)
	}
	return string(data), nil
}

func (s *Store) evictLRULocked() {
	entries, err := os.ReadDir(s.dir)
	if err != nil || len(entries) <= s.maxDocs {
		return
	}

	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var files []fileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		fullPath := filepath.Join(s.dir, e.Name())
		if info, err := e.Info(); err == nil {
			files = append(files, fileInfo{path: fullPath, modTime: info.ModTime()})
		}
	}

	if len(files) <= s.maxDocs {
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	excess := len(files) - s.maxDocs
	for i := 0; i < excess; i++ {
		_ = os.Remove(files[i].path)
	}
}
