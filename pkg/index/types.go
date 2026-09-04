package index

import (
	"time"

	"mncode/pkg/commandutil"
	"mncode/pkg/repomap"
)

const schemaVersion = 1

// Options bounds and filters the source files included in an index.
type Options struct {
	MaxFiles     int      `json:"max_files,omitempty"`
	MaxFileBytes int64    `json:"max_file_bytes,omitempty"`
	Ignore       []string `json:"ignore,omitempty"`
}

// Document stores normalized searchable metadata, never source bodies.
type Document struct {
	ID       string           `json:"id"`
	Path     string           `json:"path"`
	Language string           `json:"language"`
	Size     int64            `json:"size"`
	SHA256   string           `json:"sha256"`
	Tokens   []string         `json:"tokens"`
	Symbols  []repomap.Symbol `json:"symbols,omitempty"`
	Imports  []string         `json:"imports,omitempty"`
}

// Posting stores one document's frequency for a normalized term.
type Posting struct {
	DocumentID    string `json:"document_id"`
	TermFrequency int    `json:"term_frequency"`
}

// Index is a persisted, local-first lexical and AST-aware code index.
type Index struct {
	SchemaVersion int                   `json:"schema_version"`
	WorkspaceRoot string                `json:"workspace_root"`
	WorkspaceID   string                `json:"workspace_id"`
	BuiltAt       time.Time             `json:"built_at"`
	Options       Options               `json:"options"`
	Documents     []Document            `json:"documents"`
	Terms         map[string][]Posting  `json:"-"`
	workspace     commandutil.Workspace `json:"-"`
}

// Query controls lexical, symbol, and path-filtered search.
type Query struct {
	Text     string
	Kind     string
	PathGlob string
	Limit    int
}

// Hit is a ranked symbol or file match.
type Hit struct {
	Path      string  `json:"path"`
	Language  string  `json:"language"`
	Symbol    string  `json:"symbol,omitempty"`
	Kind      string  `json:"kind,omitempty"`
	Signature string  `json:"signature,omitempty"`
	Score     float64 `json:"score"`
	Line      int     `json:"line,omitempty"`
}

// Stats returns file, term, and symbol counts.
func (i *Index) Stats() (files, terms, symbols int) {
	if i == nil {
		return 0, 0, 0
	}
	return len(i.Documents), len(i.Terms), countSymbols(i.Documents)
}

func countSymbols(documents []Document) int {
	count := 0
	for _, document := range documents {
		count += len(document.Symbols)
	}
	return count
}
