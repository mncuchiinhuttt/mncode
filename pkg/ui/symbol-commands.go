package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/tools"
	"strings"
)

// HandleSymbolCommand searches for codebase symbols (functions, structs, classes)
func HandleSymbolCommand(parts []string, s *agent.Session) {
	if len(parts) < 2 {
		fmt.Println()
		fmt.Println(BoldCyan("AST SYMBOL SEARCH USAGE:"))
		fmt.Println("  /symbol <name>           - Search for functions, structs, interfaces, methods")
		fmt.Println()
		fmt.Println(GrayText("Example: /symbol HandleDiffCommand or /symbol Session"))
		fmt.Println()
		return
	}

	query := strings.TrimSpace(parts[1])
	symbols := tools.FindSymbolsInDir(s.WorkspaceDir, query, s.WorkspaceDir)

	if len(symbols) == 0 {
		fmt.Printf("\n%s No symbols matching '%s' found in workspace.\n\n", BoldYellow("!"), query)
		return
	}

	fmt.Printf("\n%s (%d matches for '%s'):\n", BoldPastelPink("[SEARCH] AST Symbols Found"), len(symbols), query)
	for _, sym := range symbols {
		var kindBadge string
		switch sym.Kind {
		case "func", "fn", "def":
			kindBadge = BoldCyan("FUNC     ")
		case "type", "struct":
			kindBadge = BoldGreen("STRUCT   ")
		case "interface":
			kindBadge = BoldMagenta("INTERFACE")
		case "class":
			kindBadge = BoldYellow("CLASS    ")
		default:
			kindBadge = GrayText(sym.Kind)
		}

		fmt.Printf("  [%s] %-25s %s:%d\n     %s %s\n",
			kindBadge,
			Bold(sym.Name),
			GrayText(sym.File),
			sym.Line,
			GrayText("└"),
			GrayText(sym.Signature))
	}
	fmt.Println()
}
