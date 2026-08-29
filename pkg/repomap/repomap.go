package repomap

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"mncode/pkg/artifacts"
)

const (
	defaultMaxRepoMapTokens = 1500
	approxCharsPerToken     = 4
)

// GenerateRepoMap analyzes the workspace AST, computes PageRank centrality,
// and returns a high-density, token-budgeted architectural map of the repository.
func GenerateRepoMap(workspaceDir string, maxTokens int) (string, error) {
	if workspaceDir == "" {
		workspaceDir = "."
	}
	if maxTokens <= 0 {
		maxTokens = defaultMaxRepoMapTokens
	}
	maxChars := maxTokens * approxCharsPerToken

	var fileNodes []*FileNode

	// Walk workspace and parse source files
	err := filepath.WalkDir(workspaceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, relErr := filepath.Rel(workspaceDir, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		node, parseErr := ParseSourceFile(path, rel)
		if parseErr == nil && node != nil && len(node.Symbols) > 0 {
			fileNodes = append(fileNodes, node)
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("scan workspace: %w", err)
	}

	if len(fileNodes) == 0 {
		return "", nil
	}

	// 1. Compute PageRank
	ComputePageRank(fileNodes)

	// 2. Sort files by PageRank descending
	sort.Slice(fileNodes, func(i, j int) bool {
		return fileNodes[i].PageRank > fileNodes[j].PageRank
	})

	// 3. Assemble token-budgeted skeleton
	var sb strings.Builder
	sb.WriteString("<repo-map>\n")

	currLen := 0
	filesIncluded := 0

	for _, node := range fileNodes {
		block := formatFileNode(node)
		if currLen+len(block) > maxChars && filesIncluded > 0 {
			break
		}
		sb.WriteString(block)
		currLen += len(block)
		filesIncluded++
	}

	sb.WriteString("</repo-map>")

	// 4. Scrub any sensitive patterns before returning
	return artifacts.ScrubSecrets(sb.String()), nil
}

func formatFileNode(node *FileNode) string {
	var sb strings.Builder
	sb.WriteString(node.Path + ":\n")
	for _, sym := range node.Symbols {
		sb.WriteString(fmt.Sprintf("  %s %s\n", sym.Kind, sym.Name))
	}
	return sb.String()
}
