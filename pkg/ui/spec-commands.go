package ui

import (
	"context"
	"fmt"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/spec"
)

// HandleSpecCommand creates and checks deterministic feature contracts.
func HandleSpecCommand(parts []string, session *agent.Session) {
	if session == nil || strings.TrimSpace(session.WorkspaceDir) == "" {
		fmt.Println(BoldRed("[Spec] Workspace is required."))
		return
	}
	args, asJSON := stripJSONFlag(slashArgs(parts))
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}
	if strings.HasPrefix(sub, "-") {
		fmt.Println(BoldRed("[Spec] unknown flag " + sub))
		return
	}
	if err := validateSpecArgs(sub, args); err != nil {
		fmt.Println(BoldRed("[Spec] " + err.Error()))
		return
	}
	store, err := spec.New(session.WorkspaceDir)
	if err != nil {
		fmt.Println(BoldRed("[Spec] " + err.Error()))
		return
	}
	switch sub {
	case "new", "create":
		id, title := "feature", ""
		if len(args) > 1 {
			id = args[1]
		}
		if len(args) > 2 {
			title = strings.Join(args[2:], " ")
		}
		contract, newErr := store.NewContract(context.Background(), id, title)
		if newErr != nil {
			fmt.Println(BoldRed("[Spec] " + newErr.Error()))
			return
		}
		if saveErr := store.Save(context.Background(), contract); saveErr != nil {
			fmt.Println(BoldRed("[Spec] " + saveErr.Error()))
			return
		}
		fmt.Printf("%s Contract %s created at .mncode/spec/%s.json\n", BoldGreen("[Spec OK]"), contract.ID, contract.ID)
	case "check", "matrix":
		contract, loadErr := loadSpecArg(store, args[1:])
		if loadErr != nil {
			fmt.Println(BoldRed("[Spec] " + loadErr.Error()))
			return
		}
		matrix, checkErr := store.Check(context.Background(), contract)
		if checkErr != nil {
			fmt.Println(BoldRed("[Spec] " + checkErr.Error()))
			return
		}
		if asJSON {
			printJSON(matrix)
		} else {
			printSpecMatrix(matrix)
		}
	case "show":
		contract, loadErr := loadSpecArg(store, args[1:])
		if loadErr != nil {
			fmt.Println(BoldRed("[Spec] " + loadErr.Error()))
			return
		}
		if asJSON {
			printJSON(contract)
			return
		}
		fmt.Printf("\n%s %s v%d\n  %s\n  invariants=%d cases=%d\n\n", BoldCyan("[Spec]"), contract.ID, contract.Version, contract.Title, len(contract.Invariants), len(contract.Cases))
	case "export":
		contract, loadErr := loadSpecArg(store, args[1:])
		if loadErr != nil {
			fmt.Println(BoldRed("[Spec] " + loadErr.Error()))
			return
		}
		destination := ""
		if len(args) > 2 {
			destination = args[2]
		}
		path, exportErr := store.Export(context.Background(), contract, destination)
		if exportErr != nil {
			fmt.Println(BoldRed("[Spec] " + exportErr.Error()))
			return
		}
		fmt.Printf("%s Exported: %s\n", BoldGreen("[Spec OK]"), path)
	default:
		fmt.Println("\n" + BoldCyan("SPEC COMMANDS:"))
		fmt.Println("  /spec new [id] [title]          - create a versioned contract")
		fmt.Println("  /spec check|matrix [id|path]    - run deterministic cases")
		fmt.Println("  /spec show [id|path]            - inspect contract")
		fmt.Println("  /spec export [id|path] [dest]   - export private JSON")
	}
}

func loadSpecArg(store *spec.Store, args []string) (spec.Contract, error) {
	if len(args) == 0 {
		return spec.Contract{}, fmt.Errorf("spec id or path is required")
	}
	return store.Load(context.Background(), args[0])
}
func stripJSONFlag(args []string) ([]string, bool) {
	out := make([]string, 0, len(args))
	asJSON := false
	for _, arg := range args {
		if arg == "--json" {
			asJSON = true
		} else {
			out = append(out, arg)
		}
	}
	return out, asJSON
}
func printSpecMatrix(matrix spec.Matrix) {
	fmt.Printf("\n%s %s pass=%d fail=%d skipped=%d invalid=%d\n", BoldCyan("[Spec Matrix]"), matrix.ContractID, matrix.Passed, matrix.Failed, matrix.Skipped, matrix.Invalid)
	for _, result := range matrix.Results {
		fmt.Printf("  %-18s %-8s %s\n", result.CaseID, result.Status, result.Message)
	}
	fmt.Println()
}
func validateSpecArgs(sub string, args []string) error {
	switch sub {
	case "check", "matrix", "show":
		if len(args) != 2 {
			return fmt.Errorf("%s requires exactly one spec id or path", sub)
		}
	case "export":
		if len(args) < 2 || len(args) > 3 {
			return fmt.Errorf("export requires a spec id and optional destination")
		}
	case "new", "create":
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "--") {
				return fmt.Errorf("unknown flag %q", arg)
			}
		}
	}
	return nil
}
