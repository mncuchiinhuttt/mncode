package repomap

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	tsFuncRe   = regexp.MustCompile(`export\s+(?:async\s+)?function\s+([A-Za-z0-9_]+)\s*\((.*?)\)`)
	tsClassRe  = regexp.MustCompile(`export\s+class\s+([A-Za-z0-9_]+)`)
	tsTypeRe   = regexp.MustCompile(`export\s+(?:type|interface)\s+([A-Za-z0-9_]+)`)
	pyFuncRe   = regexp.MustCompile(`def\s+([A-Za-z0-9_]+)\s*\((.*?)\):`)
	pyClassRe  = regexp.MustCompile(`class\s+([A-Za-z0-9_]+)(?:\(.*?\))?:`)
	callIdentRe = regexp.MustCompile(`\b([A-Z][a-zA-Z0-9_]{2,})\b`)
)

// ParseSourceFile extracts declarations and reference identifiers from a file.
func ParseSourceFile(path, relPath string) (*FileNode, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return parseGoFile(path, relPath)
	case ".ts", ".tsx", ".js", ".jsx":
		return parseTSFile(path, relPath)
	case ".py":
		return parsePyFile(path, relPath)
	default:
		return nil, nil
	}
}

func parseGoFile(path, relPath string) (*FileNode, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var symbols []Symbol
	var refs []string

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() {
				sig := d.Name.Name + "()"
				line := fset.Position(d.Pos()).Line
				symbols = append(symbols, Symbol{
					Name:      d.Name.Name,
					Kind:      KindFunc,
					File:      relPath,
					Line:      line,
					Signature: sig,
				})
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.IsExported() {
					kind := KindType
					if _, isStruct := ts.Type.(*ast.StructType); isStruct {
						kind = KindStruct
					} else if _, isInterface := ts.Type.(*ast.InterfaceType); isInterface {
						kind = KindInterface
					}
					line := fset.Position(ts.Pos()).Line
					symbols = append(symbols, Symbol{
						Name:      ts.Name.Name,
						Kind:      kind,
						File:      relPath,
						Line:      line,
						Signature: "type " + ts.Name.Name,
					})
				}
			}
		}
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && ident.IsExported() {
			refs = append(refs, ident.Name)
		}
		return true
	})

	return &FileNode{
		Path:    relPath,
		Symbols: symbols,
		Refs:    refs,
	}, nil
}

func parseTSFile(path, relPath string) (*FileNode, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var symbols []Symbol
	var refs []string
	scanner := bufio.NewScanner(file)
	line := 1

	for scanner.Scan() {
		text := scanner.Text()
		if m := tsFuncRe.FindStringSubmatch(text); len(m) > 1 {
			symbols = append(symbols, Symbol{Name: m[1], Kind: KindFunc, File: relPath, Line: line, Signature: m[0]})
		} else if m := tsClassRe.FindStringSubmatch(text); len(m) > 1 {
			symbols = append(symbols, Symbol{Name: m[1], Kind: KindClass, File: relPath, Line: line, Signature: m[0]})
		} else if m := tsTypeRe.FindStringSubmatch(text); len(m) > 1 {
			symbols = append(symbols, Symbol{Name: m[1], Kind: KindType, File: relPath, Line: line, Signature: m[0]})
		}
		for _, call := range callIdentRe.FindAllString(text, -1) {
			refs = append(refs, call)
		}
		line++
	}

	return &FileNode{Path: relPath, Symbols: symbols, Refs: refs}, nil
}

func parsePyFile(path, relPath string) (*FileNode, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var symbols []Symbol
	var refs []string
	scanner := bufio.NewScanner(file)
	line := 1

	for scanner.Scan() {
		text := scanner.Text()
		if m := pyFuncRe.FindStringSubmatch(text); len(m) > 1 {
			symbols = append(symbols, Symbol{Name: m[1], Kind: KindFunc, File: relPath, Line: line, Signature: m[0]})
		} else if m := pyClassRe.FindStringSubmatch(text); len(m) > 1 {
			symbols = append(symbols, Symbol{Name: m[1], Kind: KindClass, File: relPath, Line: line, Signature: m[0]})
		}
		for _, call := range callIdentRe.FindAllString(text, -1) {
			refs = append(refs, call)
		}
		line++
	}

	return &FileNode{Path: relPath, Symbols: symbols, Refs: refs}, nil
}
