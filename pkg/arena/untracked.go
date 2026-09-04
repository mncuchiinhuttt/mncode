package arena

import (
	"context"
	"fmt"
	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func appendUntracked(ctx context.Context, root commandutil.Workspace, source *Source, maxBytes int64) error {
	out, _, err := commandutil.RunBounded(ctx, root.Root, []string{"git", "status", "--porcelain=v1", "--untracked-files=all"}, commandutil.Limits{Timeout: 10 * time.Second, MaxOutputBytes: 64 * 1024})
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		rel := filepath.ToSlash(strings.TrimSpace(strings.TrimPrefix(line, "?? ")))
		if rel == "" || strings.HasPrefix(rel, ".mncode/") || secretPath(rel) {
			continue
		}
		path, err := tools.ResolveWorkspacePath(root.Root, rel, false)
		if err != nil {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxBytes {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		data = []byte(RedactDiff(string(data)))
		patch := fmt.Sprintf("\n--- /dev/null\n+++ b/%s\n+%s\n", rel, strings.ReplaceAll(string(data), "\n", "\n+"))
		if int64(len(source.Diff)+len(patch)) > maxBytes {
			return fmt.Errorf("diff exceeds %d bytes including untracked files", maxBytes)
		}
		source.Diff += patch
		source.ChangedFiles = append(source.ChangedFiles, rel)
	}
	sort.Strings(source.ChangedFiles)
	return nil
}

func secretPath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.HasPrefix(base, ".env") || strings.Contains(base, "credential") || strings.Contains(base, "secret") || strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key")
}
