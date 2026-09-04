package ui

import (
	"context"
	"fmt"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/provider"
	"mncode/pkg/sandbox"
)

// HandleSandboxCommand manages private fixture copies and bounded argv runs.
func HandleSandboxCommand(parts []string, session *agent.Session) {
	if session == nil || strings.TrimSpace(session.WorkspaceDir) == "" {
		fmt.Println(BoldRed("[Sandbox] Workspace is required."))
		return
	}
	harness, err := sandbox.New(session.WorkspaceDir)
	if err != nil {
		fmt.Println(BoldRed("[Sandbox] " + err.Error()))
		return
	}
	args := slashArgs(parts)
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}
	switch sub {
	case "init", "new":
		if len(args) > 2 {
			fmt.Println(BoldRed("[Sandbox] init accepts at most one id."))
			return
		}
		id := "default"
		if len(args) > 1 {
			id = args[1]
		}
		fixture, initErr := harness.Init(context.Background(), id)
		if initErr != nil {
			fmt.Println(BoldRed("[Sandbox] " + initErr.Error()))
			return
		}
		fmt.Printf("\n%s Fixture %s created. Edit .mncode/sandbox/fixtures/%s/fixture.json and add fixture files.\n  Command: %s\n\n", BoldGreen("[Sandbox OK]"), fixture.ID, fixture.ID, strings.Join(fixture.Command, " "))
	case "list", "ls", "":
		if len(args) > 1 {
			fmt.Println(BoldRed("[Sandbox] list accepts no arguments."))
			return
		}
		fixtures, listErr := harness.List(context.Background())
		if listErr != nil {
			fmt.Println(BoldRed("[Sandbox] " + listErr.Error()))
			return
		}
		if len(fixtures) == 0 {
			fmt.Println(GrayText("\n[Sandbox] No fixtures. Use /sandbox init <id>.\n"))
			return
		}
		fmt.Println("\n" + BoldCyan("SANDBOX FIXTURES:"))
		for _, fixture := range fixtures {
			fmt.Printf("  %-18s %s\n", BoldYellow(fixture.ID), fixture.Description)
		}
		fmt.Println()
	case "run", "test":
		fixtureID, runArgs, keep, parseErr := parseSandboxRun(args[1:])
		if parseErr != nil {
			fmt.Println(BoldRed("[Sandbox] " + parseErr.Error()))
			return
		}
		if sub == "test" && fixtureID == "" {
			fixtureID = "default"
		}
		if fixtureID == "" {
			fmt.Println(BoldYellow("Usage: /sandbox run <fixture-id> [--keep] [-- args]"))
			return
		}
		result, runErr := harness.Run(context.Background(), sandbox.RunRequest{FixtureID: fixtureID, Args: runArgs, Keep: keep})
		if runErr != nil {
			fmt.Println(BoldRed("[Sandbox] " + runErr.Error()))
			return
		}
		statusSuffix := ""
		if result.TimedOut {
			statusSuffix += " " + BoldYellow("[TIMED OUT]")
		}
		if result.Truncated {
			statusSuffix += " " + BoldYellow("[TRUNCATED]")
		}
		fmt.Printf("\n%s %s (exit %d)%s\n", BoldCyan("[Sandbox Run]"), result.ID, result.ExitCode, statusSuffix)
		if result.Stdout != "" {
			fmt.Printf("stdout:\n%s\n", result.Stdout)
		}
		if result.Stderr != "" {
			fmt.Printf("stderr:\n%s\n", result.Stderr)
		}
		if result.Error != "" {
			fmt.Printf("error: %s\n", result.Error)
		}
		fmt.Println()
	case "view":
		if len(args) != 2 {
			fmt.Println(BoldYellow("Usage: /sandbox view <run-id>"))
			return
		}
		result, viewErr := harness.View(context.Background(), args[1])
		if viewErr != nil {
			fmt.Println(BoldRed("[Sandbox] " + viewErr.Error()))
			return
		}
		printJSON(result)
	case "clean":
		if len(args) != 2 {
			fmt.Println(BoldYellow("Usage: /sandbox clean <run-id>"))
			return
		}
		if !confirmCommand(session, "clean_sandbox_run", args[1]) {
			return
		}
		if cleanErr := harness.Clean(context.Background(), args[1], true); cleanErr != nil {
			fmt.Println(BoldRed("[Sandbox] " + cleanErr.Error()))
			return
		}
		fmt.Println(BoldGreen("[Sandbox OK] Run cleaned."))
	default:
		fmt.Println("\n" + BoldCyan("SANDBOX COMMANDS:"))
		fmt.Println("  /sandbox init <id>                - create a fixture manifest")
		fmt.Println("  /sandbox list                     - list fixtures")
		fmt.Println("  /sandbox run <id> [-- args]       - run argv in a temporary copy")
		fmt.Println("  /sandbox test [id]                - run the default test fixture")
		fmt.Println("  /sandbox view|clean <run-id>      - inspect or remove a run")
	}
}

func parseSandboxRun(args []string) (string, []string, bool, error) {
	fixtureID, keep, separator := "", false, -1
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
		if arg == "--keep" {
			keep = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return "", nil, false, fmt.Errorf("unknown flag %q", arg)
		}
		if fixtureID == "" {
			fixtureID = arg
		} else {
			return "", nil, false, fmt.Errorf("unexpected argument %q; use -- for fixture args", arg)
		}
	}
	var extra []string
	if separator >= 0 {
		extra = args[separator+1:]
	}
	return fixtureID, extra, keep, nil
}

func confirmCommand(session *agent.Session, name, id string) bool {
	if session == nil || session.UI == nil {
		fmt.Println(GrayText("[Sandbox] Interactive approval is required."))
		return false
	}
	return session.UI.ConfirmToolExecution(&provider.ToolCall{ID: name + "-" + id, Name: name, Arguments: map[string]interface{}{"id": id}})
}
