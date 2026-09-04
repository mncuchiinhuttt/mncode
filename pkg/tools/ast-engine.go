package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// ASTEngine handles structural pattern matching and code synthesis.
type ASTEngine struct{}

// NewASTEngine creates an AST transformation engine.
func NewASTEngine() *ASTEngine {
	return &ASTEngine{}
}

// TransformCode applies a set of structural rewrite ops to the given source text.
func (e *ASTEngine) TransformCode(source string, ops []ASTRewriteOp) (string, int, error) {
	if len(ops) == 0 || strings.TrimSpace(source) == "" {
		return source, 0, nil
	}

	current := source
	totalMatches := 0

	for _, op := range ops {
		transformed, matches, err := e.applySingleOp(current, op)
		if err != nil {
			return source, 0, err
		}
		current = transformed
		totalMatches += matches
	}

	return current, totalMatches, nil
}

func (e *ASTEngine) applySingleOp(source string, op ASTRewriteOp) (string, int, error) {
	pat := strings.TrimSpace(op.Pat)
	out := op.Out

	if pat == "" {
		return source, 0, fmt.Errorf("empty pattern in AST rewrite op")
	}

	// 1. Build regex with named capture groups from metavariables ($VAR, $$$ARGS)
	re, varNames, err := compileASTPattern(pat)
	if err != nil {
		return source, 0, err
	}

	matches := re.FindAllStringSubmatchIndex(source, -1)
	if len(matches) == 0 {
		return source, 0, nil
	}

	var sb strings.Builder
	lastIdx := 0

	for _, loc := range matches {
		sb.WriteString(source[lastIdx:loc[0]])

		// Extract captures
		captures := make(map[string]string)
		for i, name := range varNames {
			start := loc[(i+1)*2]
			end := loc[(i+1)*2+1]
			if start >= 0 && end >= 0 {
				captures[name] = source[start:end]
			}
		}

		// Substitute into template
		substituted := out
		for name, val := range captures {
			substituted = strings.ReplaceAll(substituted, "$$$"+name, val)
			substituted = strings.ReplaceAll(substituted, "$"+name, val)
		}
		sb.WriteString(substituted)
		lastIdx = loc[1]
	}
	sb.WriteString(source[lastIdx:])

	return sb.String(), len(matches), nil
}

func compileASTPattern(pat string) (*regexp.Regexp, []string, error) {
	var varNames []string
	metaTokenRe := regexp.MustCompile(`\$\$\$([A-Z0-9_]+)|\$([A-Z0-9_]+)`)

	var sb strings.Builder
	last := 0
	for _, loc := range metaTokenRe.FindAllStringSubmatchIndex(pat, -1) {
		literal := pat[last:loc[0]]
		sb.WriteString(regexp.QuoteMeta(literal))

		if loc[2] >= 0 {
			name := pat[loc[2]:loc[3]]
			varNames = append(varNames, name)
			sb.WriteString(`(?s:(.*?))`)
		} else if loc[4] >= 0 {
			name := pat[loc[4]:loc[5]]
			varNames = append(varNames, name)
			sb.WriteString(`([a-zA-Z0-9_]+)`)
		}
		last = loc[1]
	}
	sb.WriteString(regexp.QuoteMeta(pat[last:]))

	wsRe := regexp.MustCompile(`\s+`)
	patternStr := wsRe.ReplaceAllString(sb.String(), `\s+`)

	re, err := regexp.Compile(patternStr)
	if err != nil {
		return nil, nil, fmt.Errorf("compile AST pattern %q: %w", pat, err)
	}
	return re, varNames, nil
}
