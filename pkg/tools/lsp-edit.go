package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"
)

type lspTextEdit struct {
	Range   lspRange `json:"range"`
	NewText string   `json:"newText"`
}

type lspWorkspaceEdit struct {
	Changes         map[string][]lspTextEdit `json:"changes"`
	DocumentChanges json.RawMessage          `json:"documentChanges"`
}

func (s *lspServer) rename(ctx context.Context, path string, position lspPosition, newName string) (json.RawMessage, error) {
	var edit json.RawMessage
	err := s.call(ctx, "textDocument/rename", map[string]interface{}{
		"textDocument": map[string]string{"uri": pathToURI(path)}, "position": position, "newName": newName,
	}, &edit)
	return edit, err
}

func applyWorkspaceEdit(raw json.RawMessage, baseDir string) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "Rename produced no edits.", nil
	}
	var edit lspWorkspaceEdit
	if err := json.Unmarshal(raw, &edit); err != nil {
		return "", err
	}
	if len(edit.Changes) == 0 {
		if len(edit.DocumentChanges) > 0 && string(edit.DocumentChanges) != "null" {
			return "", fmt.Errorf("language server returned unsupported documentChanges rename edit")
		}
		return "Rename produced no edits.", nil
	}
	root, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	editsByPath := make(map[string][]lspTextEdit, len(edit.Changes))
	for uri, edits := range edit.Changes {
		path, err := resolveWorkspacePath(root, uriToPath(uri), false)
		if err != nil {
			return "", fmt.Errorf("rename path rejected: %w", err)
		}
		editsByPath[path] = append(editsByPath[path], edits...)
	}
	paths := make([]string, 0, len(editsByPath))
	for path := range editsByPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	releases := make([]func(), 0, len(paths))
	defer func() {
		for index := len(releases) - 1; index >= 0; index-- {
			releases[index]()
		}
	}()
	for _, path := range paths {
		releases = append(releases, acquireEditPath(path))
	}
	updates := make([]workspaceUpdate, 0, len(paths))
	for _, path := range paths {
		update, err := prepareWorkspaceUpdatePath(path, editsByPath[path])
		if err != nil {
			return "", err
		}
		updates = append(updates, update)
	}
	for index := range updates {
		if err := stageWorkspaceUpdate(&updates[index]); err != nil {
			for _, update := range updates {
				_ = os.Remove(update.temp)
			}
			return "", fmt.Errorf("stage rename edit %s: %w", updates[index].path, err)
		}
	}
	if err := commitWorkspaceUpdates(updates); err != nil {
		return "", err
	}
	return fmt.Sprintf("Rename applied across %d file(s).", len(updates)), nil
}

func applyTextEdits(content string, edits []lspTextEdit) (string, error) {
	sort.SliceStable(edits, func(i, j int) bool {
		if edits[i].Range.Start.Line != edits[j].Range.Start.Line {
			return edits[i].Range.Start.Line > edits[j].Range.Start.Line
		}
		return edits[i].Range.Start.Character > edits[j].Range.Start.Character
	})
	lines := strings.Split(content, "\n")
	for _, edit := range edits {
		if edit.Range.Start.Line < 0 || edit.Range.Start.Line >= len(lines) ||
			edit.Range.End.Line < edit.Range.Start.Line || edit.Range.End.Line >= len(lines) ||
			edit.Range.End.Line != edit.Range.Start.Line {
			return "", fmt.Errorf("invalid LSP text edit line range")
		}
		line := []rune(lines[edit.Range.Start.Line])
		start := utf16RuneIndex(line, edit.Range.Start.Character)
		end := utf16RuneIndex(line, edit.Range.End.Character)
		if end < start {
			return "", fmt.Errorf("invalid LSP text edit range")
		}
		lines[edit.Range.Start.Line] = string(line[:start]) + edit.NewText + string(line[end:])
	}
	return strings.Join(lines, "\n"), nil
}

func utf16RuneIndex(line []rune, units int) int {
	if units <= 0 {
		return 0
	}
	used := 0
	for index, runeValue := range line {
		width := len(utf16.Encode([]rune{runeValue}))
		if used+width > units {
			return index
		}
		used += width
	}
	return len(line)
}

func pathToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.ToSlash(abs)
	if len(abs) >= 2 && abs[1] == ':' {
		abs = "/" + abs
	}
	return (&url.URL{Scheme: "file", Path: abs}).String()
}

func uriToPath(uri string) string {
	parsed, err := url.Parse(uri)
	if err == nil && parsed.Scheme == "file" {
		path := parsed.Path
		if parsed.Host != "" {
			path = "//" + parsed.Host + path
		}
		if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
			path = path[1:]
		}
		return filepath.FromSlash(path)
	}
	return filepath.FromSlash(strings.TrimPrefix(uri, "file://"))
}

func languageID(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".js":
		return "javascript"
	case ".jsx":
		return "javascriptreact"
	default:
		return "plaintext"
	}
}
