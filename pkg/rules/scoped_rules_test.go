package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopedRulesGlobMatching(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-rules-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	rulesDir := filepath.Join(tempDir, ".mncode", "rules")
	if err := os.MkdirAll(rulesDir, 0700); err != nil {
		t.Fatal(err)
	}

	// 1. Frontend rule scoped to src/components/**
	frontendRule := `---
globs: ["src/components/**", "frontend/**"]
description: "Frontend UI Guidelines"
---
- Use Tailwind CSS variables only.
- Never use hardcoded px font sizes.
`
	if err := os.WriteFile(filepath.Join(rulesDir, "frontend.md"), []byte(frontendRule), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Backend rule scoped to pkg/api/**
	backendRule := `---
globs: ["pkg/api/**", "server/**"]
description: "API Security Rules"
---
- Validate all incoming JSON requests with Zod.
- Always check JWT authorization headers.
`
	if err := os.WriteFile(filepath.Join(rulesDir, "backend.md"), []byte(backendRule), 0644); err != nil {
		t.Fatal(err)
	}

	// Test Case A: Querying frontend file
	matchedFront, err := LoadScopedRules(tempDir, []string{"src/components/navbar.tsx"})
	if err != nil {
		t.Fatalf("LoadScopedRules() error = %v", err)
	}
	if len(matchedFront) != 1 || matchedFront[0].Name != "frontend" {
		t.Fatalf("expected frontend rule match, got %v", matchedFront)
	}

	// Test Case B: Querying backend file
	matchedBack, err := LoadScopedRules(tempDir, []string{"pkg/api/auth.go"})
	if err != nil {
		t.Fatalf("LoadScopedRules() error = %v", err)
	}
	if len(matchedBack) != 1 || matchedBack[0].Name != "backend" {
		t.Fatalf("expected backend rule match, got %v", matchedBack)
	}

	// Test Case C: XML Formatting
	xml := FormatScopedRulesXML(matchedFront)
	if !strings.Contains(xml, "<path-scoped-rules>") || !strings.Contains(xml, "Tailwind CSS variables") {
		t.Fatalf("unexpected XML:\n%s", xml)
	}
}
