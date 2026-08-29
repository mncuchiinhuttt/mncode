package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// state cannot leak between workspaces or turns.
type LSPTool struct {
	BaseDir string
}

// Name returns the model-facing tool name.
func (l *LSPTool) Name() string { return "lsp_tool" }

// Description explains the semantic operations supported by the tool.
func (l *LSPTool) Description() string {
	return "Use a language server for definition, references, rename, hover, or diagnostics. Prefer this over text search when cross-file symbol accuracy matters."
}

// Schema returns the JSON schema accepted by Execute.
func (l *LSPTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type": "string", "enum": []string{"definition", "references", "rename", "diagnostics", "hover"},
			},
			"file":     map[string]interface{}{"type": "string", "description": "File path relative to the workspace."},
			"line":     map[string]interface{}{"type": "integer", "minimum": 1, "description": "One-based line."},
			"column":   map[string]interface{}{"type": "integer", "minimum": 1, "description": "One-based UTF-16 column."},
			"new_name": map[string]interface{}{"type": "string", "description": "Replacement name for rename."},
		},
		"required": []string{"action", "file"},
	}
}

func languageServerFor(path string) ([]string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return []string{"gopls"}, true
	case ".ts", ".tsx", ".js", ".jsx":
		return []string{"typescript-language-server", "--stdio"}, true
	default:
		return nil, false
	}
}

// Execute performs one semantic LSP operation inside the configured workspace.
func (l *LSPTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	operationCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	ctx = operationCtx
	action, _ := args["action"].(string)
	file, _ := args["file"].(string)
	if action == "" || file == "" {
		return "", fmt.Errorf("action and file are required")
	}
	resolved, err := resolveWorkspacePath(l.BaseDir, file, false)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", fmt.Errorf("file not found: %s", file)
	}
	launch, ok := languageServerFor(resolved)
	if !ok {
		return "", fmt.Errorf("no language server configured for %s", filepath.Ext(resolved))
	}

	server, err := startLSP(ctx, launch, l.BaseDir)
	if err != nil {
		return "", fmt.Errorf("lsp launch failed (%s): %w", strings.Join(launch, " "), err)
	}
	defer server.close()
	if err := server.initialize(ctx, l.BaseDir); err != nil {
		return "", fmt.Errorf("lsp initialize failed: %w", err)
	}
	if err := server.didOpen(ctx, resolved); err != nil {
		return "", fmt.Errorf("lsp didOpen failed: %w", err)
	}

	line, lineOK := numberArg(args["line"])
	column, columnOK := numberArg(args["column"])
	if action != "diagnostics" && (!lineOK || !columnOK || line < 1 || column < 1) {
		return "", fmt.Errorf("line and column are required for %s", action)
	}
	position := lspPosition{Line: line - 1, Character: column - 1}

	switch action {
	case "definition":
		locations, err := server.locations(ctx, "textDocument/definition", resolved, position)
		if err != nil {
			return "", err
		}
		return formatLocations(locations, l.BaseDir), nil
	case "references":
		locations, err := server.references(ctx, resolved, position)
		if err != nil {
			return "", err
		}
		return formatLocations(locations, l.BaseDir), nil
	case "hover":
		return server.hover(ctx, resolved, position)
	case "diagnostics":
		return server.diagnostics(ctx, resolved)
	case "rename":
		newName, _ := args["new_name"].(string)
		if strings.TrimSpace(newName) == "" {
			return "", fmt.Errorf("new_name is required for rename")
		}
		edits, err := server.rename(ctx, resolved, position, newName)
		if err != nil {
			return "", err
		}
		return applyWorkspaceEdit(edits, l.BaseDir)
	default:
		return "", fmt.Errorf("unsupported lsp action: %s", action)
	}
}

func formatLocations(locations []lspLocation, baseDir string) string {
	if len(locations) == 0 {
		return "No results."
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "Found %d location(s):\n", len(locations))
	for _, location := range locations {
		path := uriToPath(location.URI)
		relative, err := filepath.Rel(baseDir, path)
		if err != nil {
			relative = path
		}
		fmt.Fprintf(&builder, "- %s:%d:%d\n", relative, location.Range.Start.Line+1, location.Range.Start.Character+1)
	}
	return builder.String()
}
