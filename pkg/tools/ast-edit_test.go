package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestASTEngineTransform(t *testing.T) {
	engine := NewASTEngine()

	source := `func calculateTotal(a int, b int) int {
	return a + b
}`

	ops := []ASTRewriteOp{
		{
			Pat: "func $NAME($$$ARGS) int {\n\treturn $$$BODY\n}",
			Out: "func $NAME($$$ARGS) (int, error) {\n\treturn $$$BODY, nil\n}",
		},
	}

	transformed, matches, err := engine.TransformCode(source, ops)
	if err != nil {
		t.Fatalf("TransformCode error = %v", err)
	}
	if matches != 1 {
		t.Fatalf("expected 1 match, got %d", matches)
	}
	if !strings.Contains(transformed, "(int, error)") || !strings.Contains(transformed, "return a + b, nil") {
		t.Fatalf("unexpected transformed output:\n%s", transformed)
	}
}

func TestASTEditToolStagedCommit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mncode-ast-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	filePath := filepath.Join(tempDir, "service.go")
	initialContent := "func Greet(name string) string {\n\treturn \"hello \" + name\n}"
	if err := os.WriteFile(filePath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &ASTEditTool{BaseDir: tempDir}

	// 1. Propose Action
	proposeArgs := map[string]interface{}{
		"Action": "propose",
		"Paths":  []interface{}{"service.go"},
		"Ops": []interface{}{
			map[string]interface{}{
				"Pat": "func Greet($ARG string) string {\n\treturn $$$BODY\n}",
				"Out": "func Greet($ARG string) (string, error) {\n\treturn $$$BODY, nil\n}",
			},
		},
	}
	summary, err := tool.Execute(context.Background(), proposeArgs)
	if err != nil {
		t.Fatalf("propose error = %v", err)
	}
	if !strings.Contains(summary, "Staged Proposal") {
		t.Fatalf("expected staged summary, got %q", summary)
	}

	// 2. Apply Action
	applyArgs := map[string]interface{}{"Action": "apply"}
	applyRes, err := tool.Execute(context.Background(), applyArgs)
	if err != nil {
		t.Fatalf("apply error = %v", err)
	}
	if !strings.Contains(applyRes, "committed 1 AST file rewrites") {
		t.Fatalf("expected commit message, got %q", applyRes)
	}

	// Verify file modified on disk
	updated, _ := os.ReadFile(filePath)
	if !strings.Contains(string(updated), "(string, error)") {
		t.Fatalf("file not updated on disk:\n%s", string(updated))
	}
}
