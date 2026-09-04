package drift

import (
	"path/filepath"
	"sort"
	"strings"
)

func supportedSource(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py":
		return true
	default:
		return false
	}
}

func skipDir(name string) bool {
	switch name {
	case ".git", ".mncode", "node_modules", "vendor", "dist", "build", "target", ".next":
		return true
	default:
		return false
	}
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
	if pattern == "" {
		return false
	}
	if pattern == "*" || pattern == "**" || pattern == path {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	matched, _ := filepath.Match(pattern, path)
	return matched
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			if _, ok := seen[value]; !ok {
				seen[value] = struct{}{}
				out = append(out, value)
			}
		}
	}
	sort.Strings(out)
	return out
}
