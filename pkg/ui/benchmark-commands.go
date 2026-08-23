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

// HandleBenchmarkCommand triggers automatic performance profiling and benchmark suite execution
func HandleBenchmarkCommand(parts []string, s *agent.Session) {
	fmt.Println()
	fmt.Println(BoldPastelPink("╭── [ mncode Performance & Benchmark Suite ] ────────────────────────────────╮"))
	fmt.Println("│ ⚡ Autonomous code latency, throughput & memory profiling engine           │")
	fmt.Println(BoldPastelPink("╰────────────────────────────────────────────────────────────────────────────╯"))
	fmt.Println()

	target := ""
	if len(parts) > 1 {
		target = strings.Join(parts[1:], " ")
	}

	lang, benchCmd, hasBenchFile := detectBenchmarkEnvironment(s.WorkspaceDir)

	fmt.Printf("%s Detected environment: %s\n", BoldCyan("[Env]"), Bold(lang))

	if target != "" {
		fmt.Printf("%s Target function / file: %s\n", BoldMagenta("[Target]"), Bold(target))
		prompt := fmt.Sprintf("Please benchmark and optimize the performance of '%s'. Write a dedicated micro-benchmark suite, run it to measure baseline latency/allocations, optimize the code, and present a comparative performance table (latency, throughput, memory).", target)
		_ = s.ProcessUserInput(context.Background(), prompt)
		return
	}

	if hasBenchFile && benchCmd != "" {
		fmt.Printf("%s Running existing benchmark suite (%s)...\n\n", BoldYellow("[Running]"), benchCmd)
		start := time.Now()
		cmdParts := strings.Fields(benchCmd)
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		cmd.Dir = s.WorkspaceDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("\n%s Benchmark exited with error: %v\n", BoldRed("Notice:"), err)
		} else {
			fmt.Printf("\n%s Benchmark completed in %s\n", BoldGreen("✓"), elapsed.Round(time.Millisecond))
		}
		return
	}

	// If no existing suite, let AI inspect hot paths and benchmark
	fmt.Printf("%s No existing benchmark suite found. Spawning autonomous profiler...\n\n", BoldYellow("[Profiler]"))
	prompt := "Please inspect the codebase to identify critical performance hotspots (loops, data parsing, database queries, string allocations), generate a benchmark suite, run performance profiling, and output a detailed comparison table with optimization suggestions."
	_ = s.ProcessUserInput(context.Background(), prompt)
}

func detectBenchmarkEnvironment(dir string) (lang string, benchCmd string, hasBench bool) {
	// Go
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		hasBench = hasGoBenchFiles(dir)
		return "Go", "go test -bench=. -benchmem ./...", hasBench
	}

	// Rust
	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
		return "Rust", "cargo bench", false
	}

	// Node.js / TypeScript
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		return "TypeScript / Node.js", "npm run bench", false
	}

	// Python
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil || fileExists(filepath.Join(dir, "requirements.txt")) {
		return "Python", "pytest --benchmark-only", false
	}

	return "General", "", false
}

func hasGoBenchFiles(dir string) bool {
	found := false
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, "_test.go") {
			data, _ := os.ReadFile(path)
			if strings.Contains(string(data), "Benchmark") {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
