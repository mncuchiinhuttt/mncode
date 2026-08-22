package ui

import (
	"context"
	"fmt"
	"mncode/pkg/agent"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HandleTestCommand runs project tests and optionally heals test failures
func HandleTestCommand(parts []string, s *agent.Session) {
	shouldHeal := false
	for _, p := range parts {
		if p == "--heal" || p == "-h" || p == "heal" {
			shouldHeal = true
		}
	}

	runner, args := detectTestRunner(s.WorkspaceDir)
	if runner == "" {
		fmt.Printf("\n%s No known test runner detected (go.mod, package.json, Cargo.toml, pytest)\n\n", BoldYellow("[Notice]"))
		return
	}

	fmt.Printf("\n%s Running test suite (%s %s)...\n\n", BoldCyan("🧪 [Test Runner]"), runner, strings.Join(args, " "))

	start := time.Now()
	cmd := exec.Command(runner, args...)
	cmd.Dir = s.WorkspaceDir
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)

	rawOut := string(out)
	if err == nil {
		fmt.Printf("  %s %s (%s)\n\n",
			BoldGreen("✓ 100% Tests Passed!"),
			Bold("All assertions green"),
			GrayText(fmt.Sprintf("%.2fs", elapsed.Seconds())))
		return
	}

	fmt.Printf("  %s %s (%s)\n",
		BoldRed("✗ Tests Failed!"),
		Bold("Failures detected in suite"),
		GrayText(fmt.Sprintf("%.2fs", elapsed.Seconds())))

	lines := strings.Split(strings.TrimSpace(rawOut), "\n")
	maxShow := 12
	if len(lines) > maxShow {
		lines = lines[len(lines)-maxShow:]
	}
	for _, l := range lines {
		fmt.Printf("    %s\n", GrayText(l))
	}
	fmt.Println()

	if !shouldHeal {
		fmt.Println(GrayText("  Run '/test --heal' to let mncode automatically analyze and fix test failures."))
		fmt.Println()
		return
	}

	// Auto-Heal Loop
	fmt.Printf("%s %s\n\n", BoldPastelPink("🩺 [Auto-Heal]"), Bold("Deploying autonomous fix loop..."))
	ctx := context.Background()
	healPrompt := fmt.Sprintf("Fix the following failing tests in the workspace:\n```\n%s\n```\nAnalyze the root cause, modify the code files, and make sure all tests pass.", rawOut)

	if hErr := s.ProcessUserInput(ctx, healPrompt); hErr != nil {
		fmt.Printf("\n%s Auto-heal session error: %v\n\n", BoldRed("[Error]"), hErr)
		return
	}

	// Re-run test to verify
	fmt.Printf("\n%s Verifying auto-heal fix...\n", BoldCyan("🧪 [Re-Testing]"))
	reCmd := exec.Command(runner, args...)
	reCmd.Dir = s.WorkspaceDir
	if reOut, reErr := reCmd.CombinedOutput(); reErr == nil {
		fmt.Printf("  %s %s\n\n", BoldGreen("✓ AUTO-HEAL SUCCESSFUL!"), Bold("All tests are now passing!"))
	} else {
		fmt.Printf("  %s %s\n\n", BoldYellow("! Tests still have errors:"), strings.TrimSpace(string(reOut)))
	}
}

func detectTestRunner(wsDir string) (string, []string) {
	if _, err := os.Stat(filepath.Join(wsDir, "go.mod")); err == nil {
		return "go", []string{"test", "./..."}
	}
	if _, err := os.Stat(filepath.Join(wsDir, "Cargo.toml")); err == nil {
		return "cargo", []string{"test"}
	}
	if _, err := os.Stat(filepath.Join(wsDir, "package.json")); err == nil {
		return "npm", []string{"test"}
	}
	if _, err := os.Stat(filepath.Join(wsDir, "pytest.ini")); err == nil {
		return "pytest", []string{}
	}
	if _, err := os.Stat(filepath.Join(wsDir, "requirements.txt")); err == nil {
		return "pytest", []string{}
	}
	return "", nil
}
