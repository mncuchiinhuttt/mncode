package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/commandutil"
	"mncode/pkg/drift"
)

// HandleDriftCommand captures or checks a structural workspace baseline.
func HandleDriftCommand(parts []string, session *agent.Session) {
	if session == nil || strings.TrimSpace(session.WorkspaceDir) == "" {
		fmt.Println(BoldRed("[Drift] Workspace is required."))
		return
	}
	sub, policyPath, strict, asJSON, err := parseDriftArgs(slashArgs(parts))
	if err != nil {
		fmt.Println(BoldRed("[Drift] " + err.Error()))
		return
	}
	policy, loadedPath, err := drift.LoadPolicy(session.WorkspaceDir, policyPath)
	if err != nil {
		fmt.Println(BoldRed("[Drift] " + err.Error()))
		return
	}
	if loadedPath != "" {
		policyPath = loadedPath
	}
	sentinel, err := drift.New(session.WorkspaceDir, policy)
	if err != nil {
		fmt.Println(BoldRed("[Drift] " + err.Error()))
		return
	}
	ctx := context.Background()
	switch sub {
	case "init":
		if existing, err := sentinel.Load(); err == nil && len(existing.Files) > 0 {
			fmt.Printf("\n%s A baseline already exists (%s; %d files). Use %s to review or %s to update.\n\n",
				BoldYellow("[Drift]"), existing.ID, len(existing.Files), BoldCyan("/drift diff"), BoldCyan("/drift accept"))
			return
		}
		baseline, captureErr := sentinel.Capture(ctx)
		if captureErr != nil {
			fmt.Println(BoldRed("[Drift] " + captureErr.Error()))
			return
		}
		if err := sentinel.Save(baseline); err != nil {
			fmt.Println(BoldRed("[Drift] " + err.Error()))
			return
		}
		if asJSON {
			printJSON(baseline)
			return
		}
		fmt.Printf("\n%s Baseline %s saved (%d source files).\n\n", BoldGreen("[Drift OK]"), Bold(baseline.ID), len(baseline.Files))
	case "accept":
		if !confirmCommand(session, "accept_drift_baseline", "baseline") {
			return
		}
		baseline, captureErr := sentinel.Capture(ctx)
		if captureErr != nil {
			fmt.Println(BoldRed("[Drift] " + captureErr.Error()))
			return
		}
		if err := sentinel.Save(baseline); err != nil {
			fmt.Println(BoldRed("[Drift] " + err.Error()))
			return
		}
		if asJSON {
			printJSON(baseline)
			return
		}
		fmt.Printf("\n%s Baseline %s accepted and saved (%d source files).\n\n", BoldGreen("[Drift OK]"), Bold(baseline.ID), len(baseline.Files))
	case "check", "diff":
		baseline, loadErr := sentinel.Load()
		if loadErr != nil {
			fmt.Println(BoldRed("[Drift] " + loadErr.Error()))
			return
		}
		report, checkErr := sentinel.Check(ctx, baseline)
		if checkErr != nil {
			fmt.Println(BoldRed("[Drift] " + checkErr.Error()))
			return
		}
		if asJSON {
			printJSON(report)
		} else {
			printDriftReport(report, strict)
		}
	case "show":
		baseline, loadErr := sentinel.Load()
		if loadErr != nil {
			fmt.Println(BoldRed("[Drift] " + loadErr.Error()))
			return
		}
		if asJSON {
			printJSON(baseline)
			return
		}
		fmt.Printf("\n%s %s\n  Files: %d\n  Created: %s\n  Policy: %s\n\n", BoldCyan("[Drift Baseline]"), baseline.ID, len(baseline.Files), baseline.CreatedAt.Format("2006-01-02 15:04:05 UTC"), policyPathOrDefault(policyPath))
	default:
		fmt.Println("\n" + BoldCyan("DRIFT COMMANDS:"))
		fmt.Println("  /drift init|accept              - capture a structural baseline")
		fmt.Println("  /drift check|diff [--strict]   - compare current source against it")
		fmt.Println("  /drift show                     - show baseline metadata")
		fmt.Println("  --policy PATH --json            - select policy/output format")
	}
}

func parseDriftArgs(args []string) (string, string, bool, bool, error) {
	sub, policy, strict, asJSON := "", "", false, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--policy=") {
			policy = strings.TrimPrefix(arg, "--policy=")
			continue
		}
		switch arg {
		case "--strict":
			strict = true
		case "--json":
			asJSON = true
		case "--policy":
			if i+1 >= len(args) {
				return "", "", false, false, fmt.Errorf("--policy requires a path")
			}
			i++
			policy = args[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", false, false, fmt.Errorf("unknown flag %q", arg)
			}
			if sub == "" {
				sub = strings.ToLower(arg)
			} else {
				return "", "", false, false, fmt.Errorf("unexpected argument %q", arg)
			}
		}
	}
	if sub == "" {
		sub = "help"
	}
	return sub, policy, strict, asJSON, nil
}
func policyPathOrDefault(path string) string {
	if path == "" {
		return ".mncode/drift/policy.json (or defaults)"
	}
	return path
}
func printDriftReport(report drift.Report, strict bool) {
	code := report.ExitCode(strict)
	if !report.Drifted {
		fmt.Printf("\n%s No architectural drift detected.\n\n", BoldGreen("[Drift OK]"))
		return
	}
	fmt.Printf("\n%s %d finding(s), %d changed file(s)\n", BoldYellow("[Drift Findings]"), len(report.Findings), report.ChangedFiles)
	for _, finding := range report.Findings {
		fmt.Printf("  %-8s %-18s %-35s %s\n", finding.Severity, finding.Code, finding.Path, finding.Message)
	}
	if code != 0 {
		fmt.Println(BoldRed("  Verdict: BLOCK"))
	} else {
		fmt.Println(BoldGreen("  Verdict: WARN ONLY"))
	}
	fmt.Println()
}
func printJSON(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Println(BoldRed("[JSON] " + err.Error()))
		return
	}
	fmt.Println(commandutil.Scrub(string(data)))
}
func slashArgs(parts []string) []string {
	if len(parts) <= 1 {
		return nil
	}
	return parts[1:]
}
