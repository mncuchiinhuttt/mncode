package ui

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/provider"
)

// HandleUndoCommand previews and, after explicit approval, reverts the latest turn.
func HandleUndoCommand(parts []string, s *agent.Session) {
	plan, err := s.PreviewRollbackLastTurn()
	if err != nil {
		fmt.Printf("\n%s %v\n\n", BoldRed("[Undo Error]"), err)
		return
	}
	if !confirmRollback(plan, s) {
		fmt.Printf("\n%s Rollback cancelled; no files or history were changed.\n\n", GrayText("[Undo]"))
		return
	}
	msg, err := s.RollbackLastTurn(true)
	if err != nil {
		fmt.Printf("\n%s %v\n\n", BoldRed("[Undo Error]"), err)
		return
	}
	fmt.Println()
	fmt.Printf("  %s %s\n", BoldGreen("[OK]"), Bold(msg))
	fmt.Printf("  %s %s\n", GrayText("Status:"), GrayText("Workspace files restored and last turn removed from active memory."))
	fmt.Println()
}

// HandleRewindCommand rewinds N turns, previewing and approving every rollback.
func HandleRewindCommand(parts []string, s *agent.Session) {
	turns := 1
	if len(parts) > 1 {
		if val, err := strconv.Atoi(parts[1]); err == nil && val > 0 {
			turns = val
		}
	}

	fmt.Printf("\n%s Rewinding %d turn(s)...\n", BoldPastelPink("⏪ [Rewind]"), turns)
	completed := 0
	for i := range turns {
		plan, err := s.PreviewRollbackLastTurn()
		if err != nil {
			fmt.Printf("  %s Stopped at step %d: %v\n", BoldYellow("!"), i+1, err)
			break
		}
		if !confirmRollback(plan, s) {
			fmt.Printf("  %s Stopped at step %d: rollback cancelled.\n", BoldYellow("!"), i+1)
			break
		}
		if _, err := s.RollbackLastTurn(true); err != nil {
			fmt.Printf("  %s Stopped at step %d: %v\n", BoldYellow("!"), i+1, err)
			break
		}
		completed++
	}
	if completed == turns {
		fmt.Printf("  %s %s\n\n", BoldGreen("[OK]"), Bold(fmt.Sprintf("Rewound %d turn(s) successfully.", completed)))
	} else {
		fmt.Printf("  %s Rewound %d of %d turn(s).\n\n", BoldYellow("!"), completed, turns)
	}
}

func confirmRollback(plan *agent.RollbackPlan, s *agent.Session) bool {
	if plan == nil {
		return false
	}
	fmt.Printf("\n%s Checkpoint %s preview:\n", BoldYellow("[Rollback Preview]"), Bold(plan.CheckpointID))
	printRollbackPaths("Restore", plan.Restore)
	printRollbackPaths("Remove", plan.Remove)
	if len(plan.Skipped) > 0 {
		printRollbackPaths("Unchanged", plan.Skipped)
	}
	if len(plan.Conflicts) > 0 {
		printRollbackPaths("Conflicts", plan.Conflicts)
		fmt.Println(GrayText("Rollback is blocked because workspace changes would be overwritten."))
		return false
	}
	if s == nil || s.UI == nil {
		fmt.Println(GrayText("Rollback requires an interactive approval."))
		return false
	}
	return s.UI.ConfirmToolExecution(&provider.ToolCall{
		ID:   "checkpoint-rollback-" + plan.CheckpointID,
		Name: "rollback_checkpoint",
		Arguments: map[string]interface{}{
			"checkpoint_id": plan.CheckpointID,
			"restore":       append([]string(nil), plan.Restore...),
			"remove":        append([]string(nil), plan.Remove...),
		},
	})
}

func printRollbackPaths(label string, paths []string) {
	if len(paths) == 0 {
		return
	}
	fmt.Printf("  %s: %s\n", GrayText(label), strings.Join(paths, ", "))
}

func checkpointCreateArgs(args []string) (string, []string, error) {
	summary := "Manual checkpoint"
	if len(args) == 0 {
		return summary, nil, nil
	}
	marker := -1
	inlineMarker := -1
	inlinePaths := make([]string, 0)
	for i, arg := range args {
		switch {
		case arg == "--" || arg == "--path" || arg == "--paths":
			marker = i
		case strings.HasPrefix(arg, "--path="):
			if inlineMarker < 0 {
				inlineMarker = i
			}
			path := strings.TrimPrefix(arg, "--path=")
			if strings.TrimSpace(path) == "" {
				return "", nil, fmt.Errorf("checkpoint --path requires a path")
			}
			inlinePaths = append(inlinePaths, path)
		}
		if marker >= 0 {
			break
		}
	}
	if marker < 0 && inlineMarker >= 0 {
		summary = strings.TrimSpace(strings.Join(args[:inlineMarker], " "))
		if summary == "" {
			summary = "Manual checkpoint"
		}
		for _, arg := range args[inlineMarker:] {
			if !strings.HasPrefix(arg, "--path=") {
				return "", nil, fmt.Errorf("checkpoint path input is ambiguous; use /checkpoint create <name> --paths <file> [file...]")
			}
		}
		return summary, inlinePaths, nil
	}
	if marker < 0 {
		return strings.Join(args, " "), nil, nil
	}
	if marker > 0 {
		summary = strings.TrimSpace(strings.Join(args[:marker], " "))
		if summary == "" {
			summary = "Manual checkpoint"
		}
	}
	paths := make([]string, 0, len(args)-marker)
	for _, path := range args[marker+1:] {
		if strings.HasPrefix(path, "--") || strings.TrimSpace(path) == "" {
			return "", nil, fmt.Errorf("checkpoint path input is ambiguous; use /checkpoint create <name> --paths <file> [file...]")
		}
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return "", nil, fmt.Errorf("checkpoint %s requires at least one path", args[marker])
	}
	return summary, paths, nil
}

// selectSafeChangedPath intentionally accepts exactly one git path. Multiple
// uncommitted paths may include unrelated user edits, so callers must name
// those paths explicitly instead of having the command claim them.
func selectSafeChangedPath(s *agent.Session) ([]string, error) {
	if s == nil || strings.TrimSpace(s.WorkspaceDir) == "" {
		return nil, fmt.Errorf("cannot select changed files without a workspace")
	}
	out, err := exec.Command("git", "-C", s.WorkspaceDir, "status", "--short", "--untracked-files=all").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cannot safely select changed files: %w", err)
	}
	var paths []string
	for _, line := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) < 4 {
			return nil, fmt.Errorf("cannot safely parse git status output")
		}
		status := line[:2]
		if strings.ContainsAny(status, "RC") || strings.Contains(line[3:], " -> ") {
			return nil, fmt.Errorf("changed-file selection is ambiguous; use explicit paths")
		}
		path := line[3:]
		if path == "" {
			return nil, fmt.Errorf("cannot safely parse git status output")
		}
		paths = append(paths, path)
	}
	if len(paths) != 1 {
		return nil, fmt.Errorf("changed-file selection is ambiguous; name explicit paths with --paths")
	}
	return paths, nil
}

// HandleCheckpointCommand manages manual and automatic snapshots
func HandleCheckpointCommand(parts []string, s *agent.Session) {
	if len(parts) > 1 {
		sub := strings.ToLower(parts[1])
		switch sub {
		case "list", "ls":
			list, _ := s.ListCheckpoints()
			if len(list) == 0 {
				fmt.Printf("\n%s No checkpoints found in .mncode/checkpoints/\n\n", GrayText("[Checkpoints]"))
				return
			}
			fmt.Printf("\n%s (%d snapshots):\n", BoldCyan("[PHOTO] Saved Checkpoints"), len(list))
			for _, cp := range list {
				fmt.Printf("  • %-20s %s (%s)\n",
					Bold(cp.ID),
					Colorize(GetCurrentTheme().Text, cp.Summary),
					GrayText(cp.Timestamp.Format("15:04:05")))
			}
			fmt.Println()
			return

		case "create", "save":
			summary, ownedPaths, err := checkpointCreateArgs(parts[2:])
			if err != nil {
				fmt.Printf("\n%s %v\n\n", BoldRed("[Error]"), err)
				return
			}
			if len(ownedPaths) == 0 {
				ownedPaths, err = selectSafeChangedPath(s)
				if err != nil {
					fmt.Printf("\n%s %v\n\n", BoldRed("[Error]"), err)
					return
				}
			}
			cp, err := s.CreateTurnCheckpoint(len(s.History), summary)
			if err == nil {
				err = s.FinalizeTurnCheckpoint(cp, ownedPaths...)
			}
			if err != nil {
				fmt.Printf("\n%s %v\n\n", BoldRed("[Error]"), err)
				return
			}
			fmt.Printf("\n%s Checkpoint created and finalized: %s (%s; %d owned path(s))\n\n",
				BoldGreen("[OK]"), Bold(cp.ID), summary, len(ownedPaths))
			return
		}
	}

	fmt.Println()
	fmt.Println(BoldCyan("CHECKPOINT COMMANDS:"))
	fmt.Println("  /checkpoint create <name> [--paths <file> ...]  - Create and finalize an owned snapshot")
	fmt.Println("  /undo                      - Revert last agent turn and file edits")
	fmt.Println("  /rewind <N>                - Rewind N previous turns")
	fmt.Println()
}
