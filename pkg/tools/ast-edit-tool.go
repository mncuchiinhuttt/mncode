package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ASTEditTool performs structural AST-aware pattern matching and two-phase commits.
type ASTEditTool struct {
	BaseDir   string
	mu        sync.Mutex
	stagedOps map[string][]ASTFileProposal
}

func (t *ASTEditTool) Name() string {
	return "ast_edit"
}

func (t *ASTEditTool) Description() string {
	return "Structural AST-aware code rewrites using pattern metavariables ($VAR, $$$ARGS). Performs two-phase staging with diff preview before commit."
}

func (t *ASTEditTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"propose", "apply", "reject"},
				"description": "Operation: 'propose' (preview diffs), 'apply' (commit to disk), 'reject' (discard)",
			},
			"Ops": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"Pat": map[string]interface{}{"type": "string", "description": "AST pattern with metavariables"},
						"Out": map[string]interface{}{"type": "string", "description": "Replacement template"},
					},
					"required": []string{"Pat", "Out"},
				},
				"description": "List of structural rewrite operations",
			},
			"Paths": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Target file paths to rewrite",
			},
		},
		"required": []string{"Action"},
	}
}

func (t *ASTEditTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	action, _ := args["Action"].(string)
	t.mu.Lock()
	if t.stagedOps == nil {
		t.stagedOps = make(map[string][]ASTFileProposal)
	}
	t.mu.Unlock()

	switch strings.ToLower(action) {
	case "propose", "preview", "":
		return t.handlePropose(args)
	case "apply", "commit", "resolve":
		return t.handleApply()
	case "reject", "discard":
		return t.handleReject()
	default:
		return "", fmt.Errorf("unknown action %q", action)
	}
}

func (t *ASTEditTool) handlePropose(args map[string]interface{}) (string, error) {
	var ops []ASTRewriteOp
	if rawOps, ok := args["Ops"].([]interface{}); ok {
		for _, o := range rawOps {
			if m, ok := o.(map[string]interface{}); ok {
				pat, _ := m["Pat"].(string)
				out, _ := m["Out"].(string)
				if pat != "" {
					ops = append(ops, ASTRewriteOp{Pat: pat, Out: out})
				}
			}
		}
	}
	if len(ops) == 0 {
		return "", fmt.Errorf("at least one rewrite op is required")
	}

	var targetPaths []string
	if rawPaths, ok := args["Paths"].([]interface{}); ok {
		for _, p := range rawPaths {
			if s, ok := p.(string); ok && strings.TrimSpace(s) != "" {
				targetPaths = append(targetPaths, s)
			}
		}
	}

	engine := NewASTEngine()
	var proposals []ASTFileProposal
	var summary strings.Builder

	for _, rel := range targetPaths {
		absPath, err := resolveWorkspacePath(t.BaseDir, rel, false)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}
		original := string(data)
		modified, matches, err := engine.TransformCode(original, ops)
		if err != nil || matches == 0 {
			continue
		}

		proposal := ASTFileProposal{
			Path:       absPath,
			Original:   original,
			Modified:   modified,
			MatchCount: matches,
		}
		proposals = append(proposals, proposal)
		summary.WriteString(fmt.Sprintf("\n[Staged Proposal: %s] (%d structural matches)\n", rel, matches))
	}

	if len(proposals) == 0 {
		return "No matching AST patterns found in specified paths.", nil
	}

	t.mu.Lock()
	t.stagedOps["current"] = proposals
	t.mu.Unlock()

	summary.WriteString("\n[Review the above proposals. Call 'ast_edit' with Action='apply' to commit or Action='reject' to cancel]")
	return summary.String(), nil
}

func (t *ASTEditTool) handleApply() (string, error) {
	t.mu.Lock()
	proposals, exists := t.stagedOps["current"]
	delete(t.stagedOps, "current")
	t.mu.Unlock()

	if !exists || len(proposals) == 0 {
		return "No staged AST proposals to apply.", nil
	}

	appliedCount := 0
	for _, p := range proposals {
		dir := filepath.Dir(p.Path)
		tmpPath := filepath.Join(dir, fmt.Sprintf(".ast-tmp-%s", filepath.Base(p.Path)))
		if err := os.WriteFile(tmpPath, []byte(p.Modified), 0644); err != nil {
			continue
		}
		if err := replaceExistingFile(tmpPath, p.Path); err != nil {
			_ = os.Remove(tmpPath)
			continue
		}
		appliedCount++
	}

	return fmt.Sprintf("Successfully committed %d AST file rewrites.", appliedCount), nil
}

func (t *ASTEditTool) handleReject() (string, error) {
	t.mu.Lock()
	delete(t.stagedOps, "current")
	t.mu.Unlock()
	return "Staged AST proposals discarded.", nil
}
