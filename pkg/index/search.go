package index

import (
	"math"
	"sort"
	"strings"

	"mncode/pkg/repomap"
)

// Search ranks documents with BM25 plus exact/prefix symbol boosts.
func (i *Index) Search(query Query) []Hit {
	if i == nil || strings.TrimSpace(query.Text) == "" {
		return nil
	}
	terms := tokenize(query.Text)
	if len(terms) == 0 {
		return nil
	}
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	avgLen := averageLength(i.Documents)
	if avgLen == 0 {
		avgLen = 1
	}
	hits := make([]Hit, 0, len(i.Documents))
	for _, doc := range i.Documents {
		if query.PathGlob != "" && !matchPath(query.PathGlob, doc.Path) {
			continue
		}
		bestSymbol := bestSymbol(doc.Symbols, terms, query.Kind)
		if query.Kind != "" && bestSymbol == nil {
			continue
		}
		freq := termFrequencies(doc.Tokens)
		score := 0.0
		for _, term := range terms {
			tf := freq[term]
			if tf == 0 {
				continue
			}
			df := len(i.Terms[term])
			if df == 0 {
				continue
			}
			idf := math.Log(1 + (float64(len(i.Documents)-df)+0.5)/(float64(df)+0.5))
			lengthNorm := 1 - 0.75 + 0.75*float64(len(doc.Tokens))/avgLen
			score += idf * (float64(tf) * 2.2) / (float64(tf) + 2.2*lengthNorm)
		}
		for _, term := range terms {
			if strings.Contains(strings.ToLower(doc.Path), term) {
				score += 0.35
			}
		}
		if bestSymbol != nil {
			score += 2.0
			if symbolPrefix(bestSymbol.Name, terms) {
				score += 0.8
			}
		}
		if score == 0 {
			continue
		}
		hit := Hit{Path: doc.Path, Language: doc.Language, Score: score}
		if bestSymbol != nil {
			hit.Symbol, hit.Kind, hit.Signature, hit.Line = bestSymbol.Name, string(bestSymbol.Kind), bestSymbol.Signature, bestSymbol.Line
		}
		hits = append(hits, hit)
	}
	sort.Slice(hits, func(a, b int) bool {
		if hits[a].Score != hits[b].Score {
			return hits[a].Score > hits[b].Score
		}
		if hits[a].Path != hits[b].Path {
			return hits[a].Path < hits[b].Path
		}
		return hits[a].Line < hits[b].Line
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func averageLength(docs []Document) float64 {
	total := 0
	for _, doc := range docs {
		total += len(doc.Tokens)
	}
	if len(docs) == 0 {
		return 0
	}
	return float64(total) / float64(len(docs))
}
func termFrequencies(tokens []string) map[string]int {
	freq := make(map[string]int, len(tokens))
	for _, token := range tokens {
		freq[token]++
	}
	return freq
}
func bestSymbol(symbols []repomap.Symbol, terms []string, kind string) *repomap.Symbol {
	var best *repomap.Symbol
	bestScore := 0
	for _, symbol := range symbols {
		if kind != "" && !strings.EqualFold(kind, string(symbol.Kind)) {
			continue
		}
		name := strings.ToLower(symbol.Name)
		score := 0
		for _, term := range terms {
			if name == term {
				score += 3
			} else if strings.HasPrefix(name, term) {
				score += 1
			}
		}
		if score > bestScore {
			candidate := symbol
			best, bestScore = &candidate, score
		}
	}
	return best
}
func symbolPrefix(name string, terms []string) bool {
	lower := strings.ToLower(name)
	for _, term := range terms {
		if strings.HasPrefix(lower, term) {
			return true
		}
	}
	return false
}
