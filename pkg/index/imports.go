package index

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	indexTSImport = regexp.MustCompile(`(?:from|import)\s*[('\"]([^'\")]+)`)
	indexPyImport = regexp.MustCompile(`^\s*(?:from\s+([A-Za-z0-9_./-]+)|import\s+([A-Za-z0-9_., ]+))`)
)

func sourceImports(path string, data []byte) []string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".go" {
		file, err := parser.ParseFile(token.NewFileSet(), path, data, 0)
		if err != nil {
			return nil
		}
		imports := make([]string, 0, len(file.Imports))
		for _, spec := range file.Imports {
			imports = append(imports, strings.Trim(spec.Path.Value, `"`))
		}
		return unique(imports)
	}
	text := string(data)
	imports := make([]string, 0, 8)
	if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
		for _, match := range indexTSImport.FindAllStringSubmatch(text, -1) {
			imports = append(imports, match[1])
		}
	} else if ext == ".py" {
		for _, match := range indexPyImport.FindAllStringSubmatch(text, -1) {
			value := match[1]
			if value == "" {
				value = strings.Split(match[2], ",")[0]
			}
			imports = append(imports, strings.TrimSpace(value))
		}
	}
	return unique(imports)
}

func unique(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
