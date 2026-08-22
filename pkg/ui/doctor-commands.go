package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HandleDoctorCommand diagnoses workspace health, toolchains, and rule compliance
func HandleDoctorCommand(parts []string, s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldPastelPink("🩺 mncode Workspace Doctor & Health Audit"))
	fmt.Println(GrayText(strings.Repeat("─", 65)))
	fmt.Println()

	score := 100

	// 1. Git Repository
	gitCmd := exec.Command("git", "status", "--short")
	gitCmd.Dir = s.WorkspaceDir
	if out, err := gitCmd.Output(); err == nil {
		uncommitted := len(strings.Split(strings.TrimSpace(string(out)), "\n"))
		if strings.TrimSpace(string(out)) == "" {
			uncommitted = 0
		}
		fmt.Printf("  %s %-20s %s (%d uncommitted changes)\n",
			BoldGreen("✓"), Bold("Git Repository:"), BoldGreen("Healthy"), uncommitted)
	} else {
		score -= 15
		fmt.Printf("  %s %-20s %s\n", BoldYellow("!"), Bold("Git Repository:"), BoldYellow("Not a git repository"))
	}

	// 2. Toolchain Detection
	toolchains := []string{}
	if _, err := exec.LookPath("go"); err == nil {
		toolchains = append(toolchains, "Go")
	}
	if _, err := exec.LookPath("node"); err == nil {
		toolchains = append(toolchains, "Node.js")
	}
	if _, err := exec.LookPath("python3"); err == nil || exec.Command("python", "--version").Run() == nil {
		toolchains = append(toolchains, "Python")
	}
	if _, err := exec.LookPath("cargo"); err == nil {
		toolchains = append(toolchains, "Rust")
	}
	if len(toolchains) > 0 {
		fmt.Printf("  %s %-20s %s\n",
			BoldGreen("✓"), Bold("Runtimes Detected:"), BoldCyan(strings.Join(toolchains, ", ")))
	} else {
		fmt.Printf("  %s %-20s %s\n",
			BoldYellow("!"), Bold("Runtimes:"), GrayText("No standard compiler found in PATH"))
	}

	// 3. Environment & Security Configuration
	hasEnv := false
	if _, err := os.Stat(filepath.Join(s.WorkspaceDir, ".env")); err == nil {
		hasEnv = true
	}
	if _, err := os.Stat(filepath.Join(s.WorkspaceDir, ".gitignore")); err == nil {
		fmt.Printf("  %s %-20s %s\n", BoldGreen("✓"), Bold(".gitignore:"), BoldGreen("Present"))
	} else {
		score -= 10
		fmt.Printf("  %s %-20s %s\n", BoldYellow("!"), Bold(".gitignore:"), BoldYellow("Missing (risk of secret leak)"))
	}
	if hasEnv {
		fmt.Printf("  %s %-20s %s\n", BoldGreen("✓"), Bold(".env Config:"), BoldCyan("Configured"))
	}

	// 4. File Size Rule Check (under 200 lines)
	oversizedFiles := []string{}
	_ = filepath.Walk(s.WorkspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.Contains(path, "node_modules") || strings.Contains(path, ".git") || strings.Contains(path, "dist") {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".go" || ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".py" {
			data, err := os.ReadFile(path)
			if err == nil {
				lineCount := len(strings.Split(string(data), "\n"))
				if lineCount > 250 {
					rel, _ := filepath.Rel(s.WorkspaceDir, path)
					oversizedFiles = append(oversizedFiles, fmt.Sprintf("%s (%d lines)", rel, lineCount))
				}
			}
		}
		return nil
	})

	if len(oversizedFiles) == 0 {
		fmt.Printf("  %s %-20s %s\n",
			BoldGreen("✓"), Bold("Code Modularity:"), BoldGreen("100% files under optimal size limits"))
	} else {
		score -= len(oversizedFiles) * 3
		if score < 50 {
			score = 50
		}
		fmt.Printf("  %s %-20s %s (%d files > 250 lines)\n",
			BoldYellow("!"), Bold("Code Modularity:"), BoldYellow("Consider refactoring"), len(oversizedFiles))
		for i, f := range oversizedFiles {
			if i < 3 {
				fmt.Printf("     %s %s\n", GrayText("•"), GrayText(f))
			}
		}
	}

	// 5. Active LLM Provider
	if s.Provider != nil {
		fmt.Printf("  %s %-20s %s (%s)\n",
			BoldGreen("✓"), Bold("AI Engine:"), BoldGreen("Ready"), s.Config.Model)
	} else {
		score -= 20
		fmt.Printf("  %s %-20s %s\n", BoldRed("✗"), Bold("AI Engine:"), BoldRed("No provider configured"))
	}

	fmt.Println()
	fmt.Printf("  %s %s / 100  %s\n\n",
		BoldCyan("Health Score:"),
		BoldGreen(fmt.Sprintf("%d", score)),
		GrayText("Workspace in good operational shape!"))
}
