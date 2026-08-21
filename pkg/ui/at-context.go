package ui

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type AtOption struct {
	Tag      string // e.g. "@pkg/ui/repl.go"
	Label    string // e.g. "pkg/ui/repl.go"
	Detail   string // e.g. "file · 60 lines"
	Type     string // "file", "folder", "git", "special"
	FullPath string
}

var (
	cachedWorkspaceFiles []AtOption
	lastWorkspaceScan    time.Time
	scanMu               sync.Mutex
	atMentionRegex       = regexp.MustCompile(`@(file:|folder:|dir:)?([a-zA-Z0-9_\-\./]+|git|workspace)`)
)

// ScanWorkspaceContext crawls files & folders in the workspace
func ScanWorkspaceContext(workspaceDir string) []AtOption {
	scanMu.Lock()
	defer scanMu.Unlock()

	if time.Since(lastWorkspaceScan) < 3*time.Second && len(cachedWorkspaceFiles) > 0 {
		return cachedWorkspaceFiles
	}

	var options []AtOption

	// Built-in special context options
	options = append(options,
		AtOption{Tag: "@git", Label: "@git", Detail: "Git branch, status & diff", Type: "git"},
		AtOption{Tag: "@workspace", Label: "@workspace", Detail: "Project tree overview", Type: "special"},
		AtOption{Tag: "@file:", Label: "@file:", Detail: "Filter only files", Type: "special"},
		AtOption{Tag: "@folder:", Label: "@folder:", Detail: "Filter only directories", Type: "special"},
	)

	ignoredDirs := map[string]bool{
		".git": true, "node_modules": true, ".mncode": true,
		"dist": true, "bin": true, ".idea": true, ".vscode": true,
	}

	_ = filepath.WalkDir(workspaceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		name := d.Name()
		if d.IsDir() && ignoredDirs[name] {
			return filepath.SkipDir
		}

		rel, err := filepath.Rel(workspaceDir, path)
		if err != nil || rel == "." {
			return nil
		}

		// Skip hidden files/folders
		if strings.HasPrefix(name, ".") && name != ".env" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			options = append(options, AtOption{
				Tag:      "@" + rel + "/",
				Label:    rel + "/",
				Detail:   "folder",
				Type:     "folder",
				FullPath: path,
			})
		} else {
			info, _ := d.Info()
			sizeStr := ""
			if info != nil {
				sizeStr = formatByteSize(info.Size())
			}
			options = append(options, AtOption{
				Tag:      "@" + rel,
				Label:    rel,
				Detail:   fmt.Sprintf("file · %s", sizeStr),
				Type:     "file",
				FullPath: path,
			})
		}
		return nil
	})

	cachedWorkspaceFiles = options
	lastWorkspaceScan = time.Now()
	return options
}

func formatByteSize(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024.0)
	}
	return fmt.Sprintf("%.1f MB", float64(b)/(1024.0*1024.0))
}

// GetMatchingAtOptions searches context options based on query
func GetMatchingAtOptions(workspaceDir string, rawQuery string) []AtOption {
	all := ScanWorkspaceContext(workspaceDir)
	query := strings.TrimPrefix(rawQuery, "@")
	qLower := strings.ToLower(query)

	filterType := ""
	if strings.HasPrefix(qLower, "file:") {
		filterType = "file"
		qLower = strings.TrimPrefix(qLower, "file:")
	} else if strings.HasPrefix(qLower, "folder:") || strings.HasPrefix(qLower, "dir:") {
		filterType = "folder"
		qLower = strings.TrimPrefix(strings.TrimPrefix(qLower, "folder:"), "dir:")
	}

	if qLower == "" && filterType == "" {
		if len(all) > 15 {
			return all[:15]
		}
		return all
	}

	var matches []AtOption
	for _, opt := range all {
		if filterType != "" && opt.Type != filterType && opt.Type != "special" {
			continue
		}

		tagLower := strings.ToLower(opt.Tag)
		labelLower := strings.ToLower(opt.Label)

		if strings.Contains(tagLower, qLower) || strings.Contains(labelLower, qLower) {
			matches = append(matches, opt)
		}
	}

	if len(matches) > 15 {
		return matches[:15]
	}
	return matches
}
