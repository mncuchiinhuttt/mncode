package repomap

import (
	"strings"
)

const (
	dampingFactor  = 0.85
	maxIterations  = 20
	convergenceTol = 1e-4
)

// ComputePageRank constructs a dependency graph between files based on symbol definitions
// and references, then calculates PageRank centrality to identify key architectural files.
func ComputePageRank(files []*FileNode) {
	n := len(files)
	if n == 0 {
		return
	}

	// 1. Build map of SymbolName -> Defining File
	symbolToFile := make(map[string]int)
	for fileIdx, node := range files {
		for _, sym := range node.Symbols {
			symbolToFile[sym.Name] = fileIdx
		}
	}

	// 2. Build Adjacency Matrix
	// outgoing[i] = set of file indices that file i references
	// incoming[j] = set of file indices that reference file j
	outgoing := make([][]int, n)
	incoming := make([][]int, n)

	for srcIdx, node := range files {
		seen := make(map[int]bool)
		for _, ref := range node.Refs {
			if targetIdx, ok := symbolToFile[ref]; ok && targetIdx != srcIdx {
				if !seen[targetIdx] {
					seen[targetIdx] = true
					outgoing[srcIdx] = append(outgoing[srcIdx], targetIdx)
					incoming[targetIdx] = append(incoming[targetIdx], srcIdx)
				}
			}
		}
	}

	// 3. Initialize PageRank scores uniformly (1/N)
	ranks := make([]float64, n)
	initialRank := 1.0 / float64(n)
	for i := 0; i < n; i++ {
		ranks[i] = initialRank
	}

	// 4. Power Iteration
	for iter := 0; iter < maxIterations; iter++ {
		newRanks := make([]float64, n)
		diff := 0.0

		for j := 0; j < n; j++ {
			sum := 0.0
			for _, srcIdx := range incoming[j] {
				outDegree := len(outgoing[srcIdx])
				if outDegree > 0 {
					sum += ranks[srcIdx] / float64(outDegree)
				}
			}
			newRanks[j] = (1.0-dampingFactor)/float64(n) + dampingFactor*sum
			d := newRanks[j] - ranks[j]
			if d < 0 {
				d = -d
			}
			diff += d
		}

		ranks = newRanks
		if diff < convergenceTol {
			break
		}
	}

	// 5. Store normalized PageRank score into FileNode
	for i := 0; i < n; i++ {
		files[i].PageRank = ranks[i]
	}
}

// FormatRepoMapTree formats top-ranked files and symbols into a high-density Markdown skeleton.
func FormatRepoMapTree(files []*FileNode) string {
	var sb strings.Builder
	for _, f := range files {
		if len(f.Symbols) == 0 {
			continue
		}
		sb.WriteString(f.Path + ":\n")
		for _, s := range f.Symbols {
			sb.WriteString("  " + s.Signature + "\n")
		}
	}
	return sb.String()
}
