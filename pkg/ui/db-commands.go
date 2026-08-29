package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HandleDBCommand inspects database schemas and runs SQL queries
func HandleDBCommand(parts []string, s *agent.Session) {
	dbURL := getDatabaseURL(s.WorkspaceDir)

	if len(parts) < 2 {
		fmt.Println()
		fmt.Println(BoldCyan("DATABASE EXPLORER USAGE:"))
		fmt.Println("  /db tables               - List all tables in active database")
		fmt.Println("  /db schema <table>       - Show schema and column definitions for a table")
		fmt.Println("  /db query <SQL>          - Execute a SQL query and display formatted results")
		fmt.Println()
		if dbURL != "" {
			masked := maskDBURL(dbURL)
			fmt.Printf("  %s %s\n", GrayText("Active Database:"), BoldGreen(masked))
		} else {
			fmt.Printf("  %s %s\n", GrayText("Active Database:"), BoldYellow("None (set DATABASE_URL in .env)"))
		}
		fmt.Println()
		return
	}

	sub := strings.ToLower(parts[1])

	switch sub {
	case "tables", "list":
		if dbURL == "" {
			fmt.Printf("\n%s DATABASE_URL not found in .env\n\n", BoldRed("[Error]"))
			return
		}
		fmt.Printf("\n%s Querying tables from %s...\n\n", BoldCyan("[DB]  [Database]"), maskDBURL(dbURL))
		if strings.HasPrefix(dbURL, "postgres") || strings.HasPrefix(dbURL, "postgresql") {
			runPSQLQuery(dbURL, "\\dt")
		} else if strings.HasPrefix(dbURL, "sqlite") || strings.HasSuffix(dbURL, ".db") {
			runSQLiteQuery(dbURL, ".tables")
		} else {
			fmt.Printf("  %s Unsupported database protocol in URL\n\n", BoldYellow("!"))
		}

	case "schema", "desc":
		if len(parts) < 3 {
			fmt.Println("Usage: /db schema <table_name>")
			return
		}
		table := parts[2]
		if dbURL == "" {
			fmt.Printf("\n%s DATABASE_URL not found in .env\n\n", BoldRed("[Error]"))
			return
		}
		fmt.Printf("\n%s Schema for table '%s':\n\n", BoldCyan("[DB]  [Database Schema]"), Bold(table))
		if strings.HasPrefix(dbURL, "postgres") || strings.HasPrefix(dbURL, "postgresql") {
			runPSQLQuery(dbURL, fmt.Sprintf("\\d %s", table))
		} else if strings.HasPrefix(dbURL, "sqlite") || strings.HasSuffix(dbURL, ".db") {
			runSQLiteQuery(dbURL, fmt.Sprintf(".schema %s", table))
		}

	case "query", "sql":
		if len(parts) < 3 {
			fmt.Println("Usage: /db query \"SELECT ...\"")
			return
		}
		sqlQuery := strings.Join(parts[2:], " ")
		if dbURL == "" {
			fmt.Printf("\n%s DATABASE_URL not found in .env\n\n", BoldRed("[Error]"))
			return
		}
		fmt.Printf("\n%s Executing: %s\n\n", BoldCyan("> [SQL Query]"), Bold(sqlQuery))
		if strings.HasPrefix(dbURL, "postgres") || strings.HasPrefix(dbURL, "postgresql") {
			runPSQLQuery(dbURL, sqlQuery)
		} else if strings.HasPrefix(dbURL, "sqlite") || strings.HasSuffix(dbURL, ".db") {
			runSQLiteQuery(dbURL, sqlQuery)
		}
	}
}

func getDatabaseURL(wsDir string) string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	envPath := filepath.Join(wsDir, ".env")
	data, err := os.ReadFile(envPath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "DATABASE_URL=") {
				val := strings.TrimPrefix(l, "DATABASE_URL=")
				return strings.Trim(val, `"'`)
			}
		}
	}
	return ""
}

func maskDBURL(raw string) string {
	if len(raw) < 15 {
		return raw
	}
	parts := strings.Split(raw, "@")
	if len(parts) == 2 {
		return "postgresql://***:***@" + parts[1]
	}
	return raw[:10] + "..."
}

func runPSQLQuery(dbURL, query string) {
	if _, err := exec.LookPath("psql"); err != nil {
		fmt.Printf("  %s 'psql' CLI is not installed on PATH\n\n", BoldYellow("!"))
		return
	}
	cmd := exec.Command("psql", dbURL, "-c", query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  %s %s\n\n", BoldRed("[Query Error]:"), string(out))
		return
	}
	fmt.Printf("%s\n", string(out))
}

func runSQLiteQuery(dbPath, query string) {
	cleanPath := strings.TrimPrefix(dbPath, "sqlite://")
	cleanPath = strings.TrimPrefix(cleanPath, "sqlite:")
	if _, err := exec.LookPath("sqlite3"); err != nil {
		fmt.Printf("  %s 'sqlite3' CLI is not installed on PATH\n\n", BoldYellow("!"))
		return
	}
	cmd := exec.Command("sqlite3", cleanPath, query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  %s %s\n\n", BoldRed("[Query Error]:"), string(out))
		return
	}
	fmt.Printf("%s\n", string(out))
}
