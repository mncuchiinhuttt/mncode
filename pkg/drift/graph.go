package drift

import (
	"path/filepath"
	"sort"
	"strings"
)

func buildGraph(files []FileSnapshot) map[string]map[string]string {
	all := make(map[string]struct{}, len(files))
	for _, file := range files {
		all[file.Path] = struct{}{}
	}
	graph := make(map[string]map[string]string)
	for _, file := range files {
		for _, imported := range file.Imports {
			target := resolveImport(file.Path, imported, all)
			if target == "" {
				continue
			}
			if graph[file.Path] == nil {
				graph[file.Path] = make(map[string]string)
			}
			graph[file.Path][target] = imported
		}
	}
	return graph
}

func resolveImport(source, imported string, all map[string]struct{}) string {
	imported = strings.TrimSpace(imported)
	if imported == "" {
		return ""
	}
	if strings.HasPrefix(imported, ".") {
		base := filepath.ToSlash(filepath.Join(filepath.Dir(source), imported))
		for _, candidate := range []string{base, base + ".go", base + ".ts", base + ".tsx", base + ".js", base + ".py", filepath.Join(base, "index.ts"), filepath.Join(base, "index.js")} {
			if _, ok := all[filepath.ToSlash(candidate)]; ok {
				return filepath.ToSlash(candidate)
			}
		}
		return ""
	}
	paths := make([]string, 0, len(all))
	for path := range all {
		dir := filepath.ToSlash(filepath.Dir(path))
		withoutExt := strings.TrimSuffix(path, filepath.Ext(path))
		importedPath := strings.ReplaceAll(imported, ".", "/")
		if imported == dir || strings.HasSuffix(imported, "/"+dir) || strings.HasSuffix(importedPath, "/"+withoutExt) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

func layerFor(path string, layers []Layer) string {
	for _, layer := range layers {
		for _, pattern := range layer.Globs {
			if matchPath(pattern, path) {
				return layer.Name
			}
		}
	}
	return "unclassified"
}
func matchImport(pattern, path, layer, raw string) bool {
	pattern = strings.TrimSpace(pattern)
	return pattern == "*" || pattern == layer || matchPath(pattern, path) || pattern == raw
}

func findCycles(graph map[string]map[string]string) [][]string {
	state := make(map[string]uint8)
	stack := make([]string, 0)
	seen := make(map[string]bool)
	var cycles [][]string
	var visit func(string)
	visit = func(node string) {
		state[node] = 1
		stack = append(stack, node)
		for next := range graph[node] {
			if state[next] == 0 {
				visit(next)
				continue
			}
			if state[next] != 1 {
				continue
			}
			start := 0
			for i, item := range stack {
				if item == next {
					start = i
					break
				}
			}
			cycle := append([]string(nil), stack[start:]...)
			cycle = append(cycle, next)
			key := strings.Join(cycle, "\x00")
			if !seen[key] {
				seen[key] = true
				cycles = append(cycles, cycle)
			}
		}
		stack = stack[:len(stack)-1]
		state[node] = 2
	}
	keys := make([]string, 0, len(graph))
	for key := range graph {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if state[key] == 0 {
			visit(key)
		}
	}
	return cycles
}
