package rules

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mncode/pkg/artifacts"
)

// ScopedRule represents a development rule scoped to specific file path globs.
type ScopedRule struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Globs       []string `json:"globs"`
	Content     string   `json:"content"`
	SourcePath  string   `json:"sourcePath"`
}

// LoadScopedRules scans <workspace>/.mncode/rules/*.md and returns rules that match target files.
func LoadScopedRules(workspaceDir string, targetPaths []string) ([]ScopedRule, error) {
	if workspaceDir == "" {
		workspaceDir = "."
	}

	rulesDir := filepath.Join(workspaceDir, ".mncode", "rules")
	entries, err := os.ReadDir(rulesDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var allRules []ScopedRule
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		rulePath := filepath.Join(rulesDir, e.Name())
		rule, parseErr := parseRuleFile(rulePath, e.Name())
		if parseErr == nil && rule != nil {
			allRules = append(allRules, *rule)
		}
	}

	if len(targetPaths) == 0 {
		return allRules, nil
	}

	var matchedRules []ScopedRule
	for _, rule := range allRules {
		if matchesAnyPath(rule.Globs, targetPaths) {
			matchedRules = append(matchedRules, rule)
		}
	}

	return matchedRules, nil
}

func parseRuleFile(path, filename string) (*ScopedRule, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inFrontmatter := false
	var globs []string
	var description string
	var body strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			inFrontmatter = !inFrontmatter
			continue
		}

		if inFrontmatter {
			if strings.HasPrefix(trimmed, "globs:") {
				val := strings.TrimPrefix(trimmed, "globs:")
				val = strings.Trim(val, "[] ")
				for _, g := range strings.Split(val, ",") {
					gTrim := strings.Trim(strings.TrimSpace(g), "\"'")
					if gTrim != "" {
						globs = append(globs, gTrim)
					}
				}
			} else if strings.HasPrefix(trimmed, "description:") {
				description = strings.Trim(strings.TrimPrefix(trimmed, "description:"), "\"' ")
			}
		} else {
			body.WriteString(line + "\n")
		}
	}

	ruleName := strings.TrimSuffix(filename, ".md")
	return &ScopedRule{
		Name:        ruleName,
		Description: description,
		Globs:       globs,
		Content:     strings.TrimSpace(body.String()),
		SourcePath:  path,
	}, nil
}

func matchesAnyPath(globs []string, paths []string) bool {
	if len(globs) == 0 {
		return true // Universal rule if no globs specified
	}
	for _, p := range paths {
		pNorm := filepath.ToSlash(p)
		for _, g := range globs {
			gNorm := filepath.ToSlash(g)
			if matchGlob(gNorm, pNorm) {
				return true
			}
		}
	}
	return false
}

func matchGlob(pattern, path string) bool {
	if pattern == "*" || pattern == "**" || pattern == path {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(path, prefix) || path == prefix
	}
	matched, _ := filepath.Match(pattern, path)
	return matched
}

// FormatScopedRulesXML formats matched rules into an XML block for prompt context.
func FormatScopedRulesXML(rules []ScopedRule) string {
	if len(rules) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<path-scoped-rules>\n")
	for _, r := range rules {
		sb.WriteString(fmt.Sprintf("<!-- Rule: %s -->\n%s\n\n", r.Name, r.Content))
	}
	sb.WriteString("</path-scoped-rules>\n\n")
	return artifacts.ScrubSecrets(sb.String())
}
