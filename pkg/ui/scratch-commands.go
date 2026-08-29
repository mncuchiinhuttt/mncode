package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HandleScratchCommand manages the local code sandbox
func HandleScratchCommand(parts []string, s *agent.Session) {
	scratchDir := filepath.Join(s.WorkspaceDir, ".mncode", "scratch")
	_ = os.MkdirAll(scratchDir, 0755)

	sub := ""
	if len(parts) > 1 {
		sub = strings.ToLower(parts[1])
	}

	ext := "go"
	if sub == "ts" || sub == "js" || sub == "py" || sub == "sh" || sub == "sql" {
		ext = sub
	}

	scratchFile := filepath.Join(scratchDir, "scratch."+ext)

	switch sub {
	case "run", "eval":
		reqExt := ""
		if len(parts) > 2 {
			reqExt = strings.ToLower(parts[2])
		}
		runScratchFile(scratchDir, reqExt)
		return

	case "clear", "reset":
		_ = os.Remove(scratchFile)
		fmt.Printf("\n%s Scratchpad reset.\n\n", BoldGreen("[OK]"))
		return

	case "view", "cat":
		data, err := os.ReadFile(scratchFile)
		if err != nil {
			fmt.Printf("\n%s Scratchpad is empty.\n\n", GrayText("[Scratchpad]"))
			return
		}
		fmt.Printf("\n%s (%s):\n", BoldCyan("📝 Scratchpad Contents"), scratchFile)
		lines := strings.Split(string(data), "\n")
		for i, l := range lines {
			fmt.Printf("  %3d │ %s\n", i+1, l)
		}
		fmt.Println()
		return
	}

	// Create starter scratch template if file does not exist
	if _, err := os.Stat(scratchFile); os.IsNotExist(err) {
		starterCode := getStarterCode(ext)
		_ = os.WriteFile(scratchFile, []byte(starterCode), 0644)
	}

	fmt.Println()
	fmt.Printf("  %s %s\n", BoldGreen("[OK]"), Bold("Scratchpad Ready:"))
	fmt.Printf("    %s %s\n", BoldCyan("File:"), Bold(scratchFile))
	fmt.Printf("    %s %s\n\n", GrayText("Commands:"), GrayText("/scratch run, /scratch view, /scratch clear"))
}

func getStarterCode(ext string) string {
	switch ext {
	case "py":
		return "# mncode python scratchpad\ndef main():\n    print('Hello from Python scratchpad!')\n\nif __name__ == '__main__':\n    main()\n"
	case "ts", "js":
		return "// mncode typescript/javascript scratchpad\nconsole.log('Hello from TS scratchpad!');\n"
	case "sh":
		return "#!/bin/bash\necho 'Hello from bash scratchpad!'\n"
	default:
		return "// mncode Go scratchpad\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from Go scratchpad!\")\n}\n"
	}
}

func runScratchFile(scratchDir, reqExt string) {
	entries, _ := os.ReadDir(scratchDir)
	var target string
	var latestTime time.Time

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "scratch.") {
			if reqExt != "" && !strings.HasSuffix(e.Name(), "."+reqExt) {
				continue
			}
			info, err := e.Info()
			if err == nil && (target == "" || info.ModTime().After(latestTime)) {
				target = filepath.Join(scratchDir, e.Name())
				latestTime = info.ModTime()
			}
		}
	}
	if target == "" {
		fmt.Printf("\n%s No scratch file found. Run /scratch first.\n\n", BoldYellow("!"))
		return
	}

	ext := filepath.Ext(target)
	var cmd *exec.Cmd
	switch ext {
	case ".go":
		cmd = exec.Command("go", "run", target)
	case ".py":
		cmd = exec.Command("python3", target)
	case ".ts", ".js":
		if _, err := exec.LookPath("bun"); err == nil {
			cmd = exec.Command("bun", "run", target)
		} else {
			cmd = exec.Command("node", target)
		}
	case ".sh":
		cmd = exec.Command("bash", target)
	default:
		fmt.Printf("\n%s Unsupported scratch file type %s\n\n", BoldRed("[Error]"), ext)
		return
	}

	fmt.Printf("\n%s Running %s...\n\n", BoldCyan("> [Evaluating Scratchpad]"), filepath.Base(target))
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s %s\n\n", BoldRed("[Execution Error]:"), string(out))
		return
	}
	fmt.Printf("%s\n\n", strings.TrimSpace(string(out)))
}
