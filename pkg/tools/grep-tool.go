package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepTool searches for regex or string matches across files
type GrepTool struct {
	BaseDir string
}

func (g *GrepTool) Name() string {
	return "grep_search"
}

func (g *GrepTool) Description() string {
	return "Search for patterns in files within a directory or single file."
}

func (g *GrepTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Query": map[string]interface{}{
				"type":        "string",
				"description": "The search term or regex pattern.",
			},
			"SearchPath": map[string]interface{}{
				"type":        "string",
				"description": "The directory or file path to search.",
			},
			"CaseInsensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Perform case-insensitive search.",
			},
			"IsRegex": map[string]interface{}{
				"type":        "boolean",
				"description": "Treat Query as regular expression.",
			},
		},
		"required": []string{"Query", "SearchPath"},
	}
}

func (g *GrepTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, _ := args["Query"].(string)
	searchPath, _ := args["SearchPath"].(string)
	caseInsensitive, _ := args["CaseInsensitive"].(bool)
	isRegex, _ := args["IsRegex"].(bool)

	if query == "" || searchPath == "" {
		return "", fmt.Errorf("Query and SearchPath are required")
	}

	if !filepath.IsAbs(searchPath) && g.BaseDir != "" {
		searchPath = filepath.Join(g.BaseDir, searchPath)
	}

	var pattern *regexp.Regexp
	var err error
	patStr := query
	if !isRegex {
		patStr = regexp.QuoteMeta(query)
	}
	if caseInsensitive {
		patStr = "(?i)" + patStr
	}
	pattern, err = regexp.Compile(patStr)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	var results []string
	maxResults := 100

	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || len(results) >= maxResults {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") && name != "." && name != ".." && name != ".claude" {
				return filepath.SkipDir
			}
			if name == "node_modules" || name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only search regular files under 2MB
		if info.Size() > 2*1024*1024 {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 1
		for scanner.Scan() {
			line := scanner.Text()
			if pattern.MatchString(line) {
				results = append(results, fmt.Sprintf("%s:%d: %s", path, lineNum, line))
				if len(results) >= maxResults {
					break
				}
			}
			lineNum++
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("grep walk error: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("No matches found for '%s' in %s", query, searchPath), nil
	}

	return strings.Join(results, "\n"), nil
}
