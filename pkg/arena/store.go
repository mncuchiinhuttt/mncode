package arena

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
)

const reportDir = ".mncode/arena"

// Save persists a scrubbed report without retaining the raw diff.
func (a *Arena) Save(report Report) (string, error) {
	if a == nil {
		return "", errors.New("arena is required")
	}
	if report.Source.RepoRoot != a.Workspace.Root {
		return "", errors.New("report belongs to another workspace")
	}
	if strings.TrimSpace(report.ID) == "" || strings.ContainsAny(report.ID, `/\\`) || strings.Contains(report.ID, "..") {
		return "", errors.New("invalid arena report id")
	}
	relative := filepath.Join(reportDir, report.ID, "report.json")
	if err := a.Workspace.RejectSymlinkPath(relative); err != nil {
		return "", err
	}
	path, err := tools.ResolveWorkspacePath(a.Workspace.Root, relative, true)
	if err != nil {
		return "", err
	}
	if err := commandutil.WritePrivateJSON(path, report); err != nil {
		return "", err
	}
	return path, nil
}

// Load reads one persisted report and verifies its workspace identity.
func Load(workspace, id string) (Report, error) {
	if strings.TrimSpace(id) == "" || strings.ContainsAny(id, `/\\`) || strings.Contains(id, "..") {
		return Report{}, errors.New("invalid arena report id")
	}
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return Report{}, err
	}
	path, err := tools.ResolveWorkspacePath(root.Root, filepath.Join(reportDir, id, "report.json"), false)
	if err != nil {
		return Report{}, err
	}
	var report Report
	if err := commandutil.ReadJSON(path, &report, 4*1024*1024); err != nil {
		return Report{}, err
	}
	if report.SchemaVersion != 1 || report.ID != id || report.Source.RepoRoot != root.Root {
		return Report{}, fmt.Errorf("arena report identity mismatch")
	}
	return report, nil
}

// Delete removes a report only after explicit approval.
func Delete(ctx context.Context, workspace, id string, approved bool) error {
	if !approved {
		return errors.New("deleting arena report requires explicit approval")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" || strings.ContainsAny(id, `/\\`) || strings.Contains(id, "..") {
		return errors.New("invalid arena report id")
	}
	relative := filepath.Join(reportDir, id)
	if err := root.RejectSymlinkPath(relative); err != nil {
		return err
	}
	path, err := tools.ResolveWorkspacePath(root.Root, relative, false)
	if err != nil {
		return err
	}
	return os.RemoveAll(path)
}
