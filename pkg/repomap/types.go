package repomap

// SymbolKind represents the type of code definition.
type SymbolKind string

const (
	KindStruct    SymbolKind = "struct"
	KindInterface SymbolKind = "interface"
	KindFunc      SymbolKind = "func"
	KindType      SymbolKind = "type"
	KindClass     SymbolKind = "class"
)

// Symbol represents an exported code declaration.
type Symbol struct {
	Name      string     `json:"name"`
	Kind      SymbolKind `json:"kind"`
	File      string     `json:"file"`
	Line      int        `json:"line"`
	Signature string     `json:"signature"`
}

// FileNode represents a source file in the repository dependency graph.
type FileNode struct {
	Path     string   `json:"path"`
	Symbols  []Symbol `json:"symbols"`
	Imports  []string `json:"imports"`
	Refs     []string `json:"refs"`
	PageRank float64  `json:"pageRank"`
}
