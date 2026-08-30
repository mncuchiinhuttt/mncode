package agent

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxScanFileBytes = 2 * 1024 * 1024

type PackageInfo struct {
	Path      string
	FileCount int
	Lines     int
	Files     []string
}

type CodebaseSummary struct {
	WorkspaceDir string
	ScannedAt    time.Time
	ProjectType  string
	TotalFiles   int
	TotalLines   int
	Languages    map[string]int // Lang -> file count
	Packages     []PackageInfo
	Entrypoints  []string
	Dependencies []string
}

// ScanCodebase walks and analyzes the entire workspace architecture
func ScanCodebase(workspaceDir string) (*CodebaseSummary, error) {
	summary := &CodebaseSummary{
		WorkspaceDir: workspaceDir,
		ScannedAt:    time.Now(),
		Languages:    make(map[string]int),
		ProjectType:  "Unknown",
	}

	ignoredDirs := map[string]bool{
		".git": true, "node_modules": true, ".mncode": true,
		"dist": true, "bin": true, ".idea": true, ".vscode": true,
		"vendor": true, "build": true, "coverage": true,
	}

	pkgMap := make(map[string]*PackageInfo)

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

		if strings.HasPrefix(name, ".") && name != ".env" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxScanFileBytes {
			return nil
		}

		// Detect entrypoint
		if name == "main.go" || name == "index.ts" || name == "index.js" || name == "app.py" || name == "main.py" || name == "main.rs" {
			summary.Entrypoints = append(summary.Entrypoints, rel)
		}

		// Track language
		ext := strings.ToLower(filepath.Ext(name))
		lang := detectLang(ext, name)
		if lang != "" {
			summary.Languages[lang]++
		}

		file, err := os.Open(path)
		lineCount := 0
		if err == nil {
			content, readErr := io.ReadAll(io.LimitReader(file, maxScanFileBytes+1))
			_ = file.Close()
			if readErr == nil && int64(len(content)) <= maxScanFileBytes {
				lineCount = strings.Count(string(content), "\n") + 1
			}
		}

		summary.TotalFiles++
		summary.TotalLines += lineCount

		// Track package/directory
		dirRel := filepath.Dir(rel)
		if dirRel == "." {
			dirRel = "root"
		}
		if _, ok := pkgMap[dirRel]; !ok {
			pkgMap[dirRel] = &PackageInfo{Path: dirRel}
		}
		pkgMap[dirRel].FileCount++
		pkgMap[dirRel].Lines += lineCount
		if len(pkgMap[dirRel].Files) < 8 {
			pkgMap[dirRel].Files = append(pkgMap[dirRel].Files, name)
		}

		return nil
	})

	for _, p := range pkgMap {
		summary.Packages = append(summary.Packages, *p)
	}
	sort.Slice(summary.Packages, func(i, j int) bool {
		return summary.Packages[i].Lines > summary.Packages[j].Lines
	})

	summary.ProjectType = detectProjectType(workspaceDir, summary)
	return summary, nil
}

func detectLang(ext, name string) string {
	switch ext {
	case ".go":
		return "Go"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".js", ".jsx":
		return "JavaScript"
	case ".py":
		return "Python"
	case ".rs":
		return "Rust"
	case ".c", ".h":
		return "C"
	case ".cpp", ".hpp":
		return "C++"
	case ".md":
		return "Markdown"
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	case ".sh":
		return "Shell"
	case ".ps1":
		return "PowerShell"
	default:
		return ""
	}
}

func detectProjectType(dir string, s *CodebaseSummary) string {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		if len(s.Entrypoints) > 0 && strings.Contains(s.Entrypoints[0], "cmd") {
			return "Golang CLI Application"
		}
		return "Golang Project / Module"
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		if _, err := os.Stat(filepath.Join(dir, "next.config.js")); err == nil {
			return "Next.js Web Application"
		}
		return "Node.js / TypeScript Project"
	}
	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
		return "Rust Project"
	}
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil ||
		func() bool { _, e := os.Stat(filepath.Join(dir, "pyproject.toml")); return e == nil }() {
		return "Python Project"
	}
	return "Software Project"
}

// FormatPromptContext formats the codebase knowledge into an XML context block for the LLM
func (s *CodebaseSummary) FormatPromptContext() string {
	if s == nil || s.TotalFiles == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<codebase_architecture_map>\n")
	sb.WriteString(fmt.Sprintf("Project Type: %s\n", s.ProjectType))
	sb.WriteString(fmt.Sprintf("Scale: %d files, %d total lines of code\n", s.TotalFiles, s.TotalLines))
	if len(s.Entrypoints) > 0 {
		sb.WriteString(fmt.Sprintf("Entrypoint(s): %s\n", strings.Join(s.Entrypoints, ", ")))
	}

	var langParts []string
	for l, c := range s.Languages {
		langParts = append(langParts, fmt.Sprintf("%s (%d files)", l, c))
	}
	sb.WriteString(fmt.Sprintf("Languages: %s\n", strings.Join(langParts, ", ")))

	sb.WriteString("\nDirectory & Module Breakdown:\n")
	for i, p := range s.Packages {
		if i >= 12 {
			break
		}
		sb.WriteString(fmt.Sprintf("- %s: %d files, %d lines [%s]\n",
			p.Path, p.FileCount, p.Lines, strings.Join(p.Files, ", ")))
	}
	sb.WriteString("</codebase_architecture_map>\n")
	return sb.String()
}
