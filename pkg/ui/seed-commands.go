package ui

import (
	"bufio"
	"context"
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"
)

// HandleSeedCommand generates realistic mock data and database seed scripts
func HandleSeedCommand(parts []string, s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldPastelPink("╭── [ mncode Smart Database & Mock Data Seeder ] ───────────────────────────╮"))
	fmt.Println("│ 🎭 Autonomous relational mock data generator with localized dataset support │")
	fmt.Println(BoldPastelPink("╰────────────────────────────────────────────────────────────────────────────╯"))
	fmt.Println()

	targetTable := ""
	count := "20"
	locale := "Vietnamese"

	if len(parts) > 1 {
		targetTable = parts[1]
	}

	reader := bufio.NewReader(os.Stdin)

	if targetTable == "" {
		fmt.Print(BoldCyan("Target table or model (press Enter for all tables): "))
		ans, _ := reader.ReadString('\n')
		targetTable = strings.TrimSpace(ans)
	}

	fmt.Print(BoldCyan("Number of records to generate [default 20]: "))
	cAns, _ := reader.ReadString('\n')
	cAns = strings.TrimSpace(cAns)
	if cAns != "" {
		count = cAns
	}

	fmt.Println()
	fmt.Println(BoldYellow("Select Mock Data Locale:"))
	fmt.Println("  1. 🇻🇳  Tiếng Việt (Realistic Vietnamese names, phones, provinces, addresses)")
	fmt.Println("  2. 🌐  Global / English (International realistic data)")
	fmt.Print(BoldYellow("Enter choice [1-2] (default 1): "))
	lAns, _ := reader.ReadString('\n')
	lAns = strings.TrimSpace(lAns)
	if lAns == "2" {
		locale = "Global English"
	}

	fmt.Printf("\n%s Generating %s %s records for %s...\n\n",
		BoldGreen("[Seeder]"),
		Bold(count),
		Bold(locale),
		func() string {
			if targetTable != "" {
				return Bold(targetTable)
			}
			return "all detected database models"
		}())

	prompt := fmt.Sprintf("Please inspect the database schema (SQL migrations, schema.prisma, Drizzle schemas, or TypeScript models) in this workspace. Generate %s realistic, coherent %s relational mock records%s. Create an executable seed script (e.g. seed.sql, seed.ts, or seed.json) with proper foreign keys, realistic timestamps, and data integrity.",
		count,
		locale,
		func() string {
			if targetTable != "" {
				return fmt.Sprintf(" specifically for the '%s' table/model", targetTable)
			}
			return ""
		}())

	_ = s.ProcessUserInput(context.Background(), prompt)
}
