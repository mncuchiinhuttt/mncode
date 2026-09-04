package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os/exec"
	"strings"
)

type AuditFinding struct {
	Severity string
	Category string
	File     string
	Message  string
}

// HandleReviewCommand conducts an autonomous pre-commit code and security audit
func HandleReviewCommand(parts []string, s *agent.Session) {
	if len(parts) > 1 && strings.EqualFold(parts[1], "--arena") {
		arenaParts := append([]string{"/arena"}, parts[2:]...)
		HandleArenaCommand(arenaParts, s)
		return
	}
	diffCmd := exec.Command("git", "diff", "HEAD")
	diffCmd.Dir = s.WorkspaceDir
	out, err := diffCmd.Output()
	if err != nil {
		fmt.Printf("\n%s Failed to get git diff: %v\n\n", BoldRed("[Error]"), err)
		return
	}

	rawDiff := string(out)
	if strings.TrimSpace(rawDiff) == "" {
		fmt.Printf("\n%s Working tree is clean. No uncommitted changes to review.\n\n", BoldGreen("[OK]"))
		return
	}

	fmt.Println()
	fmt.Println(BoldPastelPink("[SECURITY]  Autonomous Pre-Commit Code & Security Review"))
	fmt.Println(GrayText(strings.Repeat("─", 65)))

	var findings []AuditFinding
	lines := strings.Split(rawDiff, "\n")
	currentFile := ""

	for _, l := range lines {
		if strings.HasPrefix(l, "+++ b/") {
			currentFile = strings.TrimPrefix(l, "+++ b/")
			continue
		}
		if !strings.HasPrefix(l, "+") || strings.HasPrefix(l, "+++") {
			continue
		}

		addedContent := l[1:]

		// Security Checks
		if strings.Contains(addedContent, "sk-") || strings.Contains(addedContent, "AIzaSy") || strings.Contains(addedContent, "ghp_") {
			findings = append(findings, AuditFinding{
				Severity: "CRITICAL",
				Category: "Secret Leak",
				File:     currentFile,
				Message:  "Potential hardcoded API token/key detected in source code.",
			})
		}
		if strings.Contains(addedContent, "SELECT * FROM") && strings.Contains(addedContent, "%s") {
			findings = append(findings, AuditFinding{
				Severity: "HIGH",
				Category: "SQL Injection",
				File:     currentFile,
				Message:  "Unparameterized SQL query with string interpolation.",
			})
		}

		// Quality Checks
		if strings.Contains(addedContent, "console.log(") || (strings.Contains(addedContent, "fmt.Println(") && !strings.Contains(currentFile, "ui/")) {
			findings = append(findings, AuditFinding{
				Severity: "INFO",
				Category: "Debug Trace",
				File:     currentFile,
				Message:  "Leftover debug print found. Consider removing before merge.",
			})
		}
		if strings.Contains(addedContent, "_ = err") || strings.Contains(addedContent, "catch (e) {}") {
			findings = append(findings, AuditFinding{
				Severity: "WARN",
				Category: "Error Handling",
				File:     currentFile,
				Message:  "Silently discarded error without logging or handling.",
			})
		}
	}

	if len(findings) == 0 {
		fmt.Println()
		fmt.Printf("  %s %s\n", BoldGreen("[OK] AUDIT PASSED [10/10]"), Bold("No security vulnerabilities or code smells detected!"))
		fmt.Printf("  %s %s\n", GrayText("Verdict:"), BoldGreen("Approved for Commit & PR"))
		fmt.Println()
		return
	}

	fmt.Printf("\n  %s %s\n\n", BoldYellow("[WARN]  Findings Detected:"), GrayText(fmt.Sprintf("(%d issues flagged)", len(findings))))
	for _, f := range findings {
		var badge string
		switch f.Severity {
		case "CRITICAL":
			badge = BoldRed("[CRITICAL]")
		case "HIGH":
			badge = BoldMagenta("[HIGH]    ")
		case "WARN":
			badge = BoldYellow("[WARN]    ")
		default:
			badge = BoldCyan("[INFO]    ")
		}
		fmt.Printf("  %s %-16s %s\n     %s %s\n",
			badge,
			Colorize(GetCurrentTheme().Primary, f.Category),
			Bold(f.File),
			GrayText("└"),
			GrayText(f.Message))
	}

	fmt.Println()
	fmt.Printf("  %s %s\n\n", BoldCyan("[INFO] Recommendation:"), GrayText("Review flagged items before running '/commit --push'."))
}
