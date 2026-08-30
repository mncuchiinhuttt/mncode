package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// IsVirtualURI reports whether the path is a virtual URI scheme.
func IsVirtualURI(path string) bool {
	p := strings.TrimSpace(path)
	return strings.HasPrefix(p, "artifact://") || strings.HasPrefix(p, "local://")
}

// ReadVirtualURI resolves and slices content from virtual URIs.
func ReadVirtualURI(uri string, workspaceDir string) (string, error) {
	uri = strings.TrimSpace(uri)
	if strings.HasPrefix(uri, "artifact://") {
		return readArtifactURI(uri)
	}
	if strings.HasPrefix(uri, "local://") {
		return readLocalURI(uri, workspaceDir)
	}
	return "", fmt.Errorf("unsupported virtual URI scheme: %s", uri)
}

func readArtifactURI(uri string) (string, error) {
	body := strings.TrimPrefix(uri, "artifact://")
	parts := strings.SplitN(body, ":", 2)
	id := parts[0]
	if !safeArtifactID(id) {
		return "", fmt.Errorf("invalid artifact id")
	}
	selector := ""
	if len(parts) > 1 {
		selector = parts[1]
	}

	content, err := GlobalStore().Get(id)
	if err != nil {
		return "", err
	}

	return sliceContent(content, selector)
}

func readLocalURI(uri string, workspaceDir string) (string, error) {
	body := strings.TrimPrefix(uri, "local://")
	parts := strings.SplitN(body, ":", 2)
	relPath := parts[0]
	selector := ""
	if len(parts) > 1 {
		selector = parts[1]
	}
	if strings.TrimSpace(workspaceDir) == "" || strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("local scratchpad path is required")
	}

	root, err := filepath.Abs(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("resolve local workspace: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve local workspace: %w", err)
	}
	scratchpadRoot, err := filepath.EvalSymlinks(filepath.Join(root, ".mncode", "scratchpad"))
	if err != nil {
		return "", fmt.Errorf("resolve local scratchpad: %w", err)
	}
	if !pathWithin(root, scratchpadRoot) {
		return "", fmt.Errorf("local scratchpad path is outside workspace")
	}
	targetPath, err := filepath.Abs(filepath.Join(scratchpadRoot, filepath.FromSlash(relPath)))
	if err != nil {
		return "", fmt.Errorf("resolve local scratchpad: %w", err)
	}
	targetPath, err = filepath.EvalSymlinks(targetPath)
	if err != nil {
		return "", fmt.Errorf("local scratchpad %q not found: %w", relPath, err)
	}
	if !pathWithin(scratchpadRoot, targetPath) {
		return "", fmt.Errorf("local scratchpad path is outside scratchpad")
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return "", fmt.Errorf("local scratchpad %q not found: %w", relPath, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("local scratchpad %q is not a regular file", relPath)
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return "", fmt.Errorf("local scratchpad %q not found: %w", relPath, err)
	}

	return sliceContent(string(data), selector)
}

func safeArtifactID(id string) bool {
	if len(id) != 8 {
		return false
	}
	for _, char := range id {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f' || char >= 'A' && char <= 'F') {
			return false
		}
	}
	return true
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func sliceContent(content string, selector string) (string, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" || selector == "raw" {
		return content, nil
	}

	lines := strings.Split(content, "\n")
	total := len(lines)

	if strings.Contains(selector, "-") {
		rangeParts := strings.SplitN(selector, "-", 2)
		start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
		end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
		if err1 != nil || err2 != nil || start < 1 || start > end {
			return content, nil
		}
		if start > total {
			return "", nil
		}
		if end > total {
			end = total
		}
		return strings.Join(lines[start-1:end], "\n"), nil
	}

	if lineNum, err := strconv.Atoi(selector); err == nil && lineNum >= 1 && lineNum <= total {
		return lines[lineNum-1], nil
	}

	return content, nil
}
