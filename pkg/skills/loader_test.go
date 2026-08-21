package skills

import (
	"path/filepath"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	raw := `---
name: sample-skill
description: A sample test skill
allowed-tools:
  - bash
  - view_file
---
# Sample Instructions
Do something great.
`
	var sk Skill
	body, err := ParseFrontmatter([]byte(raw), &sk)
	if err != nil {
		t.Fatalf("unexpected error parsing frontmatter: %v", err)
	}

	if sk.Name != "sample-skill" {
		t.Errorf("expected skill name 'sample-skill', got '%s'", sk.Name)
	}
	if len(sk.AllowedTools) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(sk.AllowedTools))
	}
	if body != "# Sample Instructions\nDo something great." {
		t.Errorf("unexpected body content: %s", body)
	}
}

func TestLoadActualClaudeCatalog(t *testing.T) {
	claudeDir := filepath.Join("..", "..", ".claude")
	catalog, err := LoadCatalog(claudeDir)
	if err != nil {
		t.Fatalf("failed to load catalog from %s: %v", claudeDir, err)
	}

	if len(catalog.Skills) == 0 {
		t.Errorf("expected to find skills in %s, got 0", claudeDir)
	}
	if len(catalog.Agents) == 0 {
		t.Errorf("expected to find agents in %s, got 0", claudeDir)
	}
	if len(catalog.Rules) == 0 {
		t.Errorf("expected to find rules in %s, got 0", claudeDir)
	}

	// Verify specific well-known skills
	if _, ok := catalog.Skills["plan"]; !ok {
		t.Errorf("expected 'plan' skill to be present")
	}
	if _, ok := catalog.Skills["research"]; !ok {
		t.Errorf("expected 'research' skill to be present")
	}

	// Verify specific agents
	if _, ok := catalog.Agents["planner"]; !ok {
		t.Errorf("expected 'planner' agent to be present")
	}
	if _, ok := catalog.Agents["code-reviewer"]; !ok {
		t.Errorf("expected 'code-reviewer' agent to be present")
	}

	rulesPrompt := catalog.FormatRules()
	if len(rulesPrompt) == 0 {
		t.Errorf("expected formatted rules to not be empty")
	}
}
