package skills

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadRules scans a rules directory and loads all markdown rules
func LoadRules(rulesDir string) (map[string]*Rule, error) {
	rules := make(map[string]*Rule)

	info, err := os.Stat(rulesDir)
	if err != nil || !info.IsDir() {
		return rules, nil
	}

	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return rules, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		ruleName := strings.TrimSuffix(entry.Name(), ".md")
		rulePath := filepath.Join(rulesDir, entry.Name())

		data, err := os.ReadFile(rulePath)
		if err != nil {
			continue
		}

		rules[ruleName] = &Rule{
			Name:     ruleName,
			Content:  string(data),
			FilePath: rulePath,
		}
	}

	return rules, nil
}
