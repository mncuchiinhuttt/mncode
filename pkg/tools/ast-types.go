package tools

// ASTRewriteOp defines a single pattern-to-replacement structural rewrite.
type ASTRewriteOp struct {
	Pat string `json:"pat"` // Structural pattern with metavariables ($VAR, $$$ARGS)
	Out string `json:"out"` // Replacement template
}

// ASTFileProposal records the staged AST modifications for a single file.
type ASTFileProposal struct {
	Path        string `json:"path"`
	Original    string `json:"original"`
	Modified    string `json:"modified"`
	UnifiedDiff string `json:"unifiedDiff"`
	MatchCount  int    `json:"matchCount"`
}

// ASTStagedSession holds in-memory staged proposals before two-phase commit.
type ASTStagedSession struct {
	ID        string            `json:"id"`
	Proposals []ASTFileProposal `json:"proposals"`
}
