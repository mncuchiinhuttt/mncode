package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"mncode/pkg/agent"
	redarena "mncode/pkg/arena"
)

// HandleArenaCommand reviews a bounded git diff with independent red-team roles.
func HandleArenaCommand(parts []string, session *agent.Session) {
	if session == nil || strings.TrimSpace(session.WorkspaceDir) == "" {
		fmt.Println(BoldRed("[Arena] Workspace is required."))
		return
	}
	base, head, model, rounds, asJSON, includeUntracked, err := parseArenaArgs(slashArgs(parts))
	if err != nil {
		fmt.Println(BoldRed("[Arena] " + err.Error()))
		return
	}
	if err := validateArenaModelForSession(session, model); err != nil {
		fmt.Println(BoldRed("[Arena] " + err.Error()))
		return
	}
	source, err := redarena.CollectSource(context.Background(), session.WorkspaceDir, base, head, includeUntracked, 512*1024)
	if err != nil {
		fmt.Println(BoldRed("[Arena] " + err.Error()))
		return
	}
	if strings.TrimSpace(source.Diff) == "" {
		if asJSON {
			printJSON(source)
			return
		}
		fmt.Printf("\n%s No diff to review (%d changed files).\n\n", BoldGreen("[Arena OK]"), len(source.ChangedFiles))
		return
	}
	if session.Provider == nil {
		if asJSON {
			printJSON(map[string]any{"schema_version": 1, "status": "unavailable", "source": source})
		} else {
			fmt.Printf("\n%s Reviewer backend unavailable. changed_files=%d diff_sha256=%s\n\n", BoldYellow("[Arena]"), len(source.ChangedFiles), source.DiffSHA256)
		}
		return
	}
	reviewer := &arenaSubagentReviewer{session: session, model: model}
	engine, err := redarena.New(session.WorkspaceDir, reviewer)
	if err != nil {
		fmt.Println(BoldRed("[Arena] " + err.Error()))
		return
	}
	report, err := engine.Review(context.Background(), source, redarena.Options{Rounds: rounds, Models: []string{model}, Timeout: 90 * time.Second, IncludeUntracked: includeUntracked})
	if err != nil {
		fmt.Println(BoldRed("[Arena] " + err.Error()))
		return
	}
	path, saveErr := engine.Save(report)
	if saveErr != nil {
		fmt.Println(BoldRed("[Arena] " + saveErr.Error()))
		return
	}
	if asJSON {
		printJSON(report)
		return
	}
	fmt.Printf("\n%s verdict=%s findings=%d\n", BoldCyan("[Arena Report]"), report.Verdict, len(report.Findings))
	for _, finding := range report.Findings {
		fmt.Printf("  %-6s %-24s %s:%d\n    %s\n", finding.Severity, finding.Category, finding.File, finding.Line, finding.Evidence)
	}
	fmt.Printf("  Report: %s\n\n", path)
}

type arenaSubagentReviewer struct {
	session *agent.Session
	model   string
	mu      sync.Mutex
}

func (r *arenaSubagentReviewer) Review(ctx context.Context, source redarena.Source, role string) ([]redarena.Finding, error) {
	instructions := map[string]string{"security adversary": "Find secrets, injection, auth, path traversal, and unsafe process behavior.", "correctness adversary": "Find logic errors, error handling gaps, races, and broken edge cases.", "maintainability adversary": "Find API compatibility, regression, observability, and operability risks."}
	prompt := fmt.Sprintf("Review this git diff as the %s. %s\nReturn only zero or more lines in this exact format:\nFINDING|severity|file|line|category|evidence|impact|recommendation\nUse severity high, medium, or low. Do not edit files or run mutating tools.\n\nDIFF:\n%s", role, instructions[role], source.Diff)
	r.mu.Lock()
	runner := &agent.SubagentRunner{ParentSession: r.session, ModelOverride: r.model, ReadOnly: true}
	r.mu.Unlock()
	output, err := runner.Run(ctx, "code-reviewer", prompt)
	if err != nil {
		return nil, err
	}
	return redarena.ParseFindings(output, role), nil
}

func parseArenaArgs(args []string) (string, string, string, int, bool, bool, error) {
	base, head, model := "", "", ""
	rounds, asJSON, include := 1, false, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "review":
		case arg == "--json":
			asJSON = true
		case arg == "--include-untracked":
			include = true
		case arg == "--base" || arg == "--head" || arg == "--model":
			if i+1 >= len(args) {
				return "", "", "", 0, false, false, fmt.Errorf("%s requires a value", arg)
			}
			i++
			if arg == "--base" {
				base = args[i]
			} else if arg == "--head" {
				head = args[i]
			} else {
				model = args[i]
			}
		case strings.HasPrefix(arg, "--base="):
			base = strings.TrimPrefix(arg, "--base=")
		case strings.HasPrefix(arg, "--head="):
			head = strings.TrimPrefix(arg, "--head=")
		case strings.HasPrefix(arg, "--model="):
			model = strings.TrimPrefix(arg, "--model=")
		case arg == "--rounds":
			if i+1 >= len(args) {
				return "", "", "", 0, false, false, fmt.Errorf("--rounds requires a number")
			}
			i++
			parsed, parseErr := strconv.Atoi(args[i])
			if parseErr != nil || parsed < 1 || parsed > 3 {
				return "", "", "", 0, false, false, fmt.Errorf("rounds must be 1-3")
			}
			rounds = parsed
		case strings.HasPrefix(arg, "--rounds="):
			parsed, parseErr := strconv.Atoi(strings.TrimPrefix(arg, "--rounds="))
			if parseErr != nil || parsed < 1 || parsed > 3 {
				return "", "", "", 0, false, false, fmt.Errorf("rounds must be 1-3")
			}
			rounds = parsed
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", "", 0, false, false, fmt.Errorf("unknown flag %q", arg)
			}
		}
	}
	if model != "" {
		if len(model) > 100 || strings.ContainsAny(model, " \t\r\n") || !knownArenaModel(model) {
			return "", "", "", 0, false, false, fmt.Errorf("invalid or unavailable model id %q", model)
		}
	}
	return base, head, model, rounds, asJSON, include, nil
}
func knownArenaModel(model string) bool {
	for _, choice := range curatedModels {
		if strings.EqualFold(choice.ID, model) {
			return true
		}
	}
	return false
}
func validateArenaModelForSession(session *agent.Session, model string) error {
	if model == "" || session == nil || session.Config == nil {
		return nil
	}
	for _, choice := range curatedModels {
		if strings.EqualFold(choice.ID, model) && choice.Provider != "" && session.Config.Provider != "" && choice.Provider != session.Config.Provider {
			return fmt.Errorf("model %q belongs to provider %q, active provider is %q", model, choice.Provider, session.Config.Provider)
		}
	}
	return nil
}
