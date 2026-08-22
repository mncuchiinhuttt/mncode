package tools

import (
	"testing"
)

func TestSymbolSearch(t *testing.T) {
	symbols := FindSymbolsInDir(".", "SymbolTool", ".")
	if len(symbols) == 0 {
		t.Fatalf("expected to find SymbolTool definition, got 0")
	}

	found := false
	for _, s := range symbols {
		if s.Name == "SymbolTool" {
			found = true
			if s.Kind != "struct" && s.Kind != "type" {
				t.Errorf("expected struct or type kind, got %s", s.Kind)
			}
			break
		}
	}
	if !found {
		t.Errorf("SymbolTool struct was not identified in search results")
	}
}
