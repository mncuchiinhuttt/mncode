package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"mncode/pkg/agent"
	codeindex "mncode/pkg/index"
)

// HandleIndexCommand builds and queries the local BM25 plus AST index.
func HandleIndexCommand(parts []string, session *agent.Session) {
	if session == nil || strings.TrimSpace(session.WorkspaceDir) == "" {
		fmt.Println(BoldRed("[Index] Workspace is required."))
		return
	}
	sub, query, kind, pathGlob, limit, asJSON, err := parseIndexArgs(slashArgs(parts))
	if err != nil {
		fmt.Println(BoldRed("[Index] " + err.Error()))
		return
	}
	ctx := context.Background()
	switch sub {
	case "build", "rebuild":
		idx, buildErr := codeindex.Build(ctx, session.WorkspaceDir, codeindex.Options{})
		if buildErr != nil {
			fmt.Println(BoldRed("[Index] " + buildErr.Error()))
			return
		}
		if saveErr := idx.Save(); saveErr != nil {
			fmt.Println(BoldRed("[Index] " + saveErr.Error()))
			return
		}
		files, terms, symbols := idx.Stats()
		fmt.Printf("\n%s Indexed %d files, %d terms, %d symbols.\n\n", BoldGreen("[Index OK]"), files, terms, symbols)
	case "show", "status":
		idx, openErr := codeindex.Open(session.WorkspaceDir)
		if openErr != nil {
			fmt.Println(BoldRed("[Index] " + openErr.Error()))
			return
		}
		files, terms, symbols := idx.Stats()
		if asJSON {
			printJSON(map[string]any{"schema_version": 1, "files": files, "terms": terms, "symbols": symbols, "built_at": idx.BuiltAt})
		} else {
			fmt.Printf("\n%s files=%d terms=%d symbols=%d built=%s\n\n", BoldCyan("[Index]"), files, terms, symbols, idx.BuiltAt.Format("2006-01-02 15:04:05 UTC"))
		}
	case "query":
		handleIndexQuery(session.WorkspaceDir, query, kind, pathGlob, limit, asJSON)
	case "clear":
		if !confirmCommand(session, "clear_code_index", "index") {
			return
		}
		if clearErr := codeindex.Clear(session.WorkspaceDir, true); clearErr != nil {
			fmt.Println(BoldRed("[Index] " + clearErr.Error()))
			return
		}
		fmt.Println(BoldGreen("[Index OK] Index cleared."))
	default:
		if sub != "help" {
			handleIndexQuery(session.WorkspaceDir, strings.TrimSpace(strings.Join(slashArgs(parts), " ")), kind, pathGlob, limit, asJSON)
			return
		}
		fmt.Println("\n" + BoldCyan("INDEX COMMANDS:"))
		fmt.Println("  /index build|rebuild              - build local persisted index")
		fmt.Println("  /index query <text> [--kind K]    - ranked BM25 + AST search")
		fmt.Println("  /index show                       - show index status")
		fmt.Println("  /index clear                      - remove index (approval)")
	}
}

func handleIndexQuery(workspace, query, kind, pathGlob string, limit int, asJSON bool) {
	idx, err := codeindex.Open(workspace)
	if err != nil {
		fmt.Println(BoldRed("[Index] " + err.Error()))
		return
	}
	hits := idx.Search(codeindex.Query{Text: query, Kind: kind, PathGlob: pathGlob, Limit: limit})
	if asJSON {
		printJSON(hits)
		return
	}
	if len(hits) == 0 {
		fmt.Println(GrayText("\n[Index] No matches.\n"))
		return
	}
	fmt.Println("\n" + BoldCyan("INDEX RESULTS:"))
	for _, hit := range hits {
		fmt.Printf("  %-8.3f %-36s %-22s line=%d\n", hit.Score, hit.Path, hit.Symbol, hit.Line)
	}
	fmt.Println()
}

func parseIndexArgs(args []string) (string, string, string, string, int, bool, error) {
	sub, kind, pathGlob, query := "", "", "", ""
	limit, asJSON := 10, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			asJSON = true
		case arg == "--kind":
			if i+1 >= len(args) {
				return "", "", "", "", 0, false, fmt.Errorf("--kind requires a value")
			}
			i++
			kind = args[i]
		case strings.HasPrefix(arg, "--kind="):
			kind = strings.TrimPrefix(arg, "--kind=")
		case arg == "--path":
			if i+1 >= len(args) {
				return "", "", "", "", 0, false, fmt.Errorf("--path requires a value")
			}
			i++
			pathGlob = args[i]
		case strings.HasPrefix(arg, "--path="):
			pathGlob = strings.TrimPrefix(arg, "--path=")
		case arg == "--limit":
			if i+1 >= len(args) {
				return "", "", "", "", 0, false, fmt.Errorf("--limit requires a number")
			}
			i++
			parsed, parseErr := strconv.Atoi(args[i])
			if parseErr != nil || parsed < 1 || parsed > 50 {
				return "", "", "", "", 0, false, fmt.Errorf("limit must be 1-50")
			}
			limit = parsed
		case strings.HasPrefix(arg, "--limit="):
			parsed, parseErr := strconv.Atoi(strings.TrimPrefix(arg, "--limit="))
			if parseErr != nil || parsed < 1 || parsed > 50 {
				return "", "", "", "", 0, false, fmt.Errorf("limit must be 1-50")
			}
			limit = parsed
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", "", "", 0, false, fmt.Errorf("unknown flag %q", arg)
			}
			if sub == "" {
				sub = strings.ToLower(arg)
			} else if query == "" {
				query = arg
			} else {
				query += " " + arg
			}
		}
	}
	if sub == "" {
		sub = "help"
	}
	if sub != "query" && sub != "build" && sub != "rebuild" && sub != "show" && sub != "status" && sub != "clear" && sub != "help" {
		query = strings.TrimSpace(strings.Join(append([]string{sub}, strings.Fields(query)...), " "))
		sub = "query"
	}
	return sub, query, kind, pathGlob, limit, asJSON, nil
}
