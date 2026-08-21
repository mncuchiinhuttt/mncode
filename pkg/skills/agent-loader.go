package skills

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadAgents scans an agents directory and parses all .md agent specifications
func LoadAgents(agentsDir string) (map[string]*Agent, error) {
	agents := make(map[string]*Agent)

	info, err := os.Stat(agentsDir)
	if err != nil || !info.IsDir() {
		return agents, nil
	}

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return agents, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		agentName := strings.TrimSuffix(entry.Name(), ".md")
		agentPath := filepath.Join(agentsDir, entry.Name())

		data, err := os.ReadFile(agentPath)
		if err != nil {
			continue
		}

		prompt := string(data)
		role := agentName
		desc := "Specialized agent for " + agentName

		// Try to extract title / role from first markdown heading
		lines := strings.Split(prompt, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") {
				role = strings.TrimPrefix(trimmed, "# ")
				break
			}
		}

		agents[agentName] = &Agent{
			Name:        agentName,
			Role:        role,
			Description: desc,
			Prompt:      prompt,
			FilePath:    agentPath,
		}
	}

	return agents, nil
}
