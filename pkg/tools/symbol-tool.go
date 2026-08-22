package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SymbolResult struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Signature string `json:"signature"`
}

type SymbolTool struct {
	WorkspaceDir string
}

func (t *SymbolTool) Name() string {
	return "find_symbol"
}

func (t *SymbolTool) Description() string {
	return "Find function, struct, interface, class, method, or type definitions across the codebase using AST symbol analysis."
}

func (t *SymbolTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Query": map[string]interface{}{
				"type":        "string",
				"description": "Symbol name or substring to search for (e.g. 'Session', 'HandleDiffCommand', 'User').",
			},
			"SearchPath": map[string]interface{}{
				"type":        "string",
				"description": "Optional subdirectory to limit search scope (defaults to workspace root).",
			},
		},
		"required": []string{"Query"},
	}
}

func (t *SymbolTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, _ := args["Query"].(string)
	if query == "" {
		return "", fmt.Errorf("query argument is required")
	}

	searchPath, _ := args["SearchPath"].(string)
	targetDir := t.WorkspaceDir
	if searchPath != "" {
		targetDir = filepath.Join(t.WorkspaceDir, searchPath)
	}

	symbols := FindSymbolsInDir(targetDir, query, t.WorkspaceDir)
	if len(symbols) == 0 {
		return fmt.Sprintf("No symbols found matching '%s'.", query), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d symbol(s) matching '%s':\n\n", len(symbols), query))
	for _, s := range symbols {
		sb.WriteString(fmt.Sprintf("- [%s] %s (%s:%d)\n  %s\n", s.Kind, s.Name, s.File, s.Line, s.Signature))
	}
	return sb.String(), nil
}

var symbolPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^(?:pub\s+)?(?:export\s+)?(?:default\s+)?(func|type|struct|interface|class|def|fn)\s+([A-Za-z0-9_]+)(.*)$`),
	regexp.MustCompile(`(?m)^func\s+\(\w+\s+\*?(\w+)\)\s+([A-Za-z0-9_]+)(.*)$`), // Go Method
}

func FindSymbolsInDir(dir, query, baseDir string) []SymbolResult {
	var results []SymbolResult
	lowerQuery := strings.ToLower(query)

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.Contains(path, "node_modules") || strings.Contains(path, ".git") || strings.Contains(path, "dist") {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".py" && ext != ".rs" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		rel, _ := filepath.Rel(baseDir, path)

		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.Contains(strings.ToLower(trimmed), lowerQuery) {
				continue
			}

			for _, pat := range symbolPatterns {
				matches := pat.FindStringSubmatch(trimmed)
				if len(matches) >= 3 {
					kind := matches[1]
					name := matches[2]
					if strings.Contains(strings.ToLower(name), lowerQuery) {
						results = append(results, SymbolResult{
							Name:      name,
							Kind:      kind,
							File:      rel,
							Line:      i + 1,
							Signature: trimmed,
						})
						if len(results) >= 25 {
							return nil
						}
						break
					}
				}
			}
		}
		return nil
	})

	return results
}
