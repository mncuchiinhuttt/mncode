package drift

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"mncode/pkg/commandutil"
	"mncode/pkg/repomap"
)

var (
	tsImportRE = regexp.MustCompile(`(?:from|import)\s*[('\"]([^'\")]+)`)
	pyImportRE = regexp.MustCompile(`^\s*(?:from\s+([A-Za-z0-9_./-]+)|import\s+([A-Za-z0-9_., ]+))`)
)

func collect(ctx context.Context, workspace commandutil.Workspace, policy Policy, limits commandutil.Limits) ([]FileSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	defaults := commandutil.DefaultLimits()
	if limits.MaxFiles <= 0 {
		limits.MaxFiles = defaults.MaxFiles
	}
	if limits.MaxFileBytes <= 0 {
		limits.MaxFileBytes = defaults.MaxFileBytes
	}
	files := make([]FileSnapshot, 0, 256)
	err := filepath.WalkDir(workspace.Root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
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
		rel, err := workspace.Relative(path)
		if err != nil || ignored(rel, policy.Ignore) || !supportedSource(path) {
			return nil
		}
		if len(files) >= limits.MaxFiles {
			return fmt.Errorf("drift scan exceeds %d files", limits.MaxFiles)
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect %s: %w", rel, err)
		}
		if info.Size() > limits.MaxFileBytes {
			return nil
		}
		snapshot, err := snapshotFile(path, rel, info.Size())
		if err != nil {
			return fmt.Errorf("snapshot %s: %w", rel, err)
		}
		files = append(files, snapshot)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func snapshotFile(path, rel string, size int64) (FileSnapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return FileSnapshot{}, err
	}
	defer file.Close()
	hash := sha256.New()
	data, err := io.ReadAll(io.LimitReader(file, size+1))
	if err != nil || int64(len(data)) > size {
		return FileSnapshot{}, fmt.Errorf("read %s", rel)
	}
	_, _ = hash.Write(data)
	node, _ := repomap.ParseSourceFile(path, rel)
	var symbols []repomap.Symbol
	if node != nil {
		symbols = append(symbols, node.Symbols...)
	}
	if filepath.Ext(path) == ".go" {
		symbols = goSignatures(path, data, rel, symbols)
	}
	return FileSnapshot{Path: rel, SHA256: hex.EncodeToString(hash.Sum(nil)), Size: size,
		Symbols: symbols, Imports: sourceImports(path, data)}, nil
}

func goSignatures(path string, data []byte, rel string, symbols []repomap.Symbol) []repomap.Symbol {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, data, 0)
	if err != nil {
		return symbols
	}
	byName := make(map[string]int, len(symbols))
	for i, symbol := range symbols {
		byName[symbol.Name] = i
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !fn.Name.IsExported() {
			continue
		}
		var typeText strings.Builder
		if err := format.Node(&typeText, fileSet, fn.Type); err != nil {
			continue
		}
		signature := "func " + fn.Name.Name + strings.TrimPrefix(typeText.String(), "func")
		if index, ok := byName[fn.Name.Name]; ok {
			symbols[index].Signature = signature
		} else {
			symbols = append(symbols, repomap.Symbol{Name: fn.Name.Name, Kind: repomap.KindFunc, File: rel, Line: fileSet.Position(fn.Pos()).Line, Signature: signature})
		}
	}
	return symbols
}

func sourceImports(path string, data []byte) []string {
	text := string(data)
	if filepath.Ext(path) == ".go" {
		file, err := parser.ParseFile(token.NewFileSet(), path, data, 0)
		if err != nil {
			return nil
		}
		imports := make([]string, 0, len(file.Imports))
		for _, spec := range file.Imports {
			imports = append(imports, strings.Trim(spec.Path.Value, `"`))
		}
		return imports
	}
	var imports []string
	if ext := filepath.Ext(path); ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
		for _, match := range tsImportRE.FindAllStringSubmatch(text, -1) {
			imports = append(imports, match[1])
		}
	} else if filepath.Ext(path) == ".py" {
		for _, match := range pyImportRE.FindAllStringSubmatch(text, -1) {
			value := match[1]
			if value == "" {
				value = strings.Split(match[2], ",")[0]
			}
			imports = append(imports, strings.TrimSpace(value))
		}
	}
	return uniqueSorted(imports)
}
