package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Checkpoint struct {
	ID        string    `json:"id"`
	TurnIndex int       `json:"turn_index"`
	Timestamp time.Time `json:"timestamp"`
	Summary   string    `json:"summary"`
	PatchFile string    `json:"patch_file"`
}

// CreateTurnCheckpoint saves a git diff snapshot before modifying files
func (s *Session) CreateTurnCheckpoint(turnIndex int, summary string) (*Checkpoint, error) {
	ckptDir := filepath.Join(s.WorkspaceDir, ".mncode", "checkpoints")
	if err := os.MkdirAll(ckptDir, 0755); err != nil {
		return nil, err
	}

	diffCmd := exec.Command("git", "diff", "HEAD")
	diffCmd.Dir = s.WorkspaceDir
	out, _ := diffCmd.Output()

	ckptID := fmt.Sprintf("ckpt-%d-%d", turnIndex, time.Now().Unix())
	patchPath := filepath.Join(ckptDir, ckptID+".patch")
	_ = os.WriteFile(patchPath, out, 0644)

	ckpt := &Checkpoint{
		ID:        ckptID,
		TurnIndex: turnIndex,
		Timestamp: time.Now(),
		Summary:   summary,
		PatchFile: patchPath,
	}

	metaBytes, _ := json.Marshal(ckpt)
	_ = os.WriteFile(filepath.Join(ckptDir, ckptID+".json"), metaBytes, 0644)

	return ckpt, nil
}

// RollbackLastTurn restores working tree to previous turn checkpoint
func (s *Session) RollbackLastTurn() (string, error) {
	if len(s.History) == 0 {
		return "", fmt.Errorf("no conversation history to undo")
	}

	// 1. Git clean/checkout changes
	revertCmd := exec.Command("git", "checkout", ".")
	revertCmd.Dir = s.WorkspaceDir
	if out, err := revertCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git checkout error: %s", string(out))
	}

	// 2. Pop history until last user prompt
	poppedCount := 0
	for len(s.History) > 0 {
		last := s.History[len(s.History)-1]
		s.History = s.History[:len(s.History)-1]
		poppedCount++
		if last.Role == "user" {
			break
		}
	}

	_ = s.Save()
	return fmt.Sprintf("Successfully reverted last turn (removed %d conversation messages and restored files)", poppedCount), nil
}

// ListCheckpoints lists saved turn checkpoints
func (s *Session) ListCheckpoints() ([]Checkpoint, error) {
	ckptDir := filepath.Join(s.WorkspaceDir, ".mncode", "checkpoints")
	entries, err := os.ReadDir(ckptDir)
	if err != nil {
		return nil, nil
	}

	var list []Checkpoint
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			data, err := os.ReadFile(filepath.Join(ckptDir, e.Name()))
			if err == nil {
				var cp Checkpoint
				if json.Unmarshal(data, &cp) == nil {
					list = append(list, cp)
				}
			}
		}
	}
	return list, nil
}
