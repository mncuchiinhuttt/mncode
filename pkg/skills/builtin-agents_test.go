package skills

import "testing"

func TestLoadCatalogIncludesBuiltInAgents(t *testing.T) {
	catalog, err := LoadCatalog("")
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}

	for _, name := range []string{
		"planner", "researcher", "scout", "tester", "debugger", "code-reviewer", "docs-manager",
	} {
		agent, ok := catalog.Agents[name]
		if !ok || agent == nil {
			t.Fatalf("built-in agent %q is missing", name)
		}
		if agent.Prompt == "" {
			t.Fatalf("built-in agent %q has an empty prompt", name)
		}
		if agent.FilePath == "" {
			t.Fatalf("built-in agent %q has no source marker", name)
		}
	}
}
