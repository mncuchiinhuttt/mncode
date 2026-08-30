package index

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"mncode/pkg/commandutil"
	"mncode/pkg/repomap"
)

// Build scans supported source files and creates a fresh local index.
func Build(ctx context.Context, workspace string, opts Options) (*Index, error) {
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	defaults := commandutil.DefaultLimits()
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = defaults.MaxFiles
	}
	if opts.MaxFileBytes <= 0 {
		opts.MaxFileBytes = defaults.MaxFileBytes
	}
	idx := &Index{SchemaVersion: schemaVersion, WorkspaceRoot: root.Root, WorkspaceID: root.Identity, BuiltAt: nowUTC(), Options: opts, Terms: make(map[string][]Posting), workspace: root}
	err = filepath.WalkDir(root.Root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if err := contextError(ctx); err != nil {
			return err
		}
		if entry.IsDir() {
			if skipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := root.Relative(path)
		if err != nil || ignored(rel, opts.Ignore) || secretPath(rel) || !supported(rel) {
			return nil
		}
		if len(idx.Documents) >= opts.MaxFiles {
			return fmt.Errorf("index scan exceeds %d files", opts.MaxFiles)
		}
		info, err := entry.Info()
		if err != nil || info.Size() > opts.MaxFileBytes {
			return nil
		}
		data, err := readBounded(path, info.Size(), opts.MaxFileBytes)
		if err != nil {
			return nil
		}
		if bytes.IndexByte(data, 0) >= 0 {
			return nil
		}
		hash := sha256.Sum256(data)
		node, _ := repomap.ParseSourceFile(path, rel)
		doc := Document{ID: rel, Path: rel, Language: language(rel), Size: info.Size(), SHA256: hex.EncodeToString(hash[:]), Tokens: tokenize(string(data)), Imports: sourceImports(rel, data)}
		if node != nil {
			doc.Symbols = node.Symbols
		}
		idx.Documents = append(idx.Documents, doc)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(idx.Documents, func(a, b int) bool { return idx.Documents[a].Path < idx.Documents[b].Path })
	idx.rebuildPostings()
	return idx, nil
}

func (i *Index) rebuildPostings() {
	i.Terms = make(map[string][]Posting)
	for _, doc := range i.Documents {
		freq := make(map[string]int)
		for _, term := range doc.Tokens {
			freq[term]++
		}
		terms := make([]string, 0, len(freq))
		for term := range freq {
			terms = append(terms, term)
		}
		sort.Strings(terms)
		for _, term := range terms {
			i.Terms[term] = append(i.Terms[term], Posting{DocumentID: doc.ID, TermFrequency: freq[term]})
		}
	}
}

func readBounded(path string, size, cap int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, cap+1))
	if err != nil || int64(len(data)) > cap || int64(len(data)) != size {
		return nil, fmt.Errorf("bounded read failed")
	}
	return data, nil
}
func tokenize(text string) []string {
	text = commandutil.Scrub(text)
	parts := strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_' })
	terms := make([]string, 0, min(len(parts), 2048))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if len(part) < 2 || len(part) > 64 {
			continue
		}
		terms = append(terms, part)
		if len(terms) >= 2048 {
			break
		}
	}
	return terms
}

func supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".java":
		return true
	}
	return false
}
func language(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}
func skipDir(name string) bool {
	switch name {
	case ".git", ".mncode", "node_modules", "vendor", "dist", "build", "target", ".next":
		return true
	}
	return false
}
func secretPath(path string) bool {
	lower := strings.ToLower(filepath.Base(path))
	return strings.HasPrefix(lower, ".env") || strings.Contains(lower, "credential") || strings.Contains(lower, "secret") || strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key")
}
func ignored(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchPath(pattern, path) {
			return true
		}
	}
	return false
}
func matchPath(pattern, path string) bool {
	pattern, path = filepath.ToSlash(strings.TrimSpace(pattern)), filepath.ToSlash(path)
	if pattern == "*" || pattern == "**" || pattern == path {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	ok, _ := filepath.Match(pattern, path)
	return ok
}
func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
func nowUTC() (t time.Time) { return time.Now().UTC() }
