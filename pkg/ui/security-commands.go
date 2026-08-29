package ui

import (
	"bufio"
	"context"
	"fmt"
	"mncode/pkg/agent"
	"os"
	"strings"
)

// HandleSecurityCommand runs deep security audit, secrets detection, and auto-patching
func HandleSecurityCommand(parts []string, s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldPastelPink("╭── [ mncode Security Sentinel & OWASP Auditor ] ───────────────────────────╮"))
	fmt.Println("│ [SECURITY] Enterprise vulnerability scanning, secrets leak detection & auto-patch  │")
	fmt.Println(BoldPastelPink("╰────────────────────────────────────────────────────────────────────────────╯"))
	fmt.Println()

	if len(parts) > 1 {
		sub := strings.ToLower(parts[1])
		switch sub {
		case "patch", "--patch", "-p", "fix", "--fix":
			runSecurityPatch(s)
			return
		case "secrets", "--secrets", "-s":
			runSecretsScan(s)
			return
		case "deps", "--deps", "cve", "--cve":
			runDependenciesAudit(s)
			return
		}
	}

	fmt.Println(BoldCyan("Select Security Action:"))
	fmt.Println("  1. [SECURITY]  Full OWASP Top 10 & Codebase Security Audit (Recommended)")
	fmt.Println("  2. [ACTION]  Auto-Patch All Detected Vulnerabilities (Self-Healing Fix)")
	fmt.Println("  3. 🔑  Scan Leaked API Keys & Hardcoded Secrets")
	fmt.Println("  4. [PACKAGE]  Audit Outdated Dependencies & Known CVEs")
	fmt.Println("  5. ❌  Cancel")
	fmt.Println()
	fmt.Print(BoldYellow("Enter choice [1-5] (default 1): "))

	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(ans)

	switch ans {
	case "2":
		runSecurityPatch(s)
	case "3":
		runSecretsScan(s)
	case "4":
		runDependenciesAudit(s)
	case "5", "q", "exit":
		fmt.Printf("\n%s Security scan cancelled.\n\n", BoldYellow("[Cancelled]"))
	default:
		runFullAudit(s)
	}
}

func runFullAudit(s *agent.Session) {
	fmt.Printf("\n%s Initiating deep OWASP Top 10 & Codebase Security Audit...\n\n", BoldCyan("[Scanning]"))
	prompt := "Please perform a thorough security audit of the workspace following OWASP Top 10 standards: inspect for SQL/Command Injections, XSS, Hardcoded API keys/passwords, SSRF, Broken Authentication/Authorization, insecure headers, and Path Traversal. Present a structured table with Severity (CRITICAL, HIGH, MEDIUM, LOW), Vulnerable File & Line, Risk Description, and Recommended Patch."
	_ = s.ProcessUserInput(context.Background(), prompt)
}

func runSecurityPatch(s *agent.Session) {
	fmt.Printf("\n%s Scanning and auto-patching critical security vulnerabilities...\n\n", BoldMagenta("[Auto-Patch]"))
	prompt := "Please identify all security vulnerabilities in the codebase (insecure inputs, missing sanitization, hardcoded secrets, injection risks) and actively apply clean, backward-compatible security patches directly to the files. Run tests to verify the patches compile and pass."
	_ = s.ProcessUserInput(context.Background(), prompt)
}

func runSecretsScan(s *agent.Session) {
	fmt.Printf("\n%s Scanning repository for hardcoded secrets, tokens, and credentials...\n\n", BoldYellow("[Secrets]"))
	prompt := "Please scan all files in the workspace for hardcoded secrets (API keys, private tokens, database passwords, OAuth client secrets, AWS/GCP credentials, JWT secrets, .env leaks). Provide remediation steps to migrate secrets to environment variables."
	_ = s.ProcessUserInput(context.Background(), prompt)
}

func runDependenciesAudit(s *agent.Session) {
	fmt.Printf("\n%s Auditing package dependencies for known CVEs and security advisories...\n\n", BoldCyan("[Dependencies]"))
	prompt := "Please audit the package dependencies (go.mod, package.json, requirements.txt, Cargo.toml) for outdated libraries, known CVEs, and security advisories. Recommend secure upgrade versions."
	_ = s.ProcessUserInput(context.Background(), prompt)
}
