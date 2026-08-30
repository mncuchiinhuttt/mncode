package drift

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"mncode/pkg/commandutil"
	"mncode/pkg/repomap"
)

// Sentinel compares the current source structure with a saved baseline.
type Sentinel struct {
	Workspace commandutil.Workspace
	Policy    Policy
	Limits    commandutil.Limits
}

// New creates a drift sentinel rooted at a canonical workspace.
func New(workspace string, policy Policy) (*Sentinel, error) {
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	return &Sentinel{Workspace: root, Policy: policy, Limits: commandutil.DefaultLimits()}, nil
}

// Capture creates a baseline without storing source bodies.
func (s *Sentinel) Capture(ctx context.Context) (Baseline, error) {
	if s == nil {
		return Baseline{}, errors.New("drift sentinel is required")
	}
	files, err := collect(ctx, s.Workspace, s.Policy, s.Limits)
	if err != nil {
		return Baseline{}, err
	}
	return Baseline{SchemaVersion: 1, ID: commandutil.NewID("baseline"), WorkspaceRoot: s.Workspace.Root,
		WorkspaceID: s.Workspace.Identity, ToolVersion: "mncode-drift-v1", CreatedAt: time.Now().UTC(), Policy: s.Policy, Files: files}, nil
}

// Check compares a baseline against the current source tree.
func (s *Sentinel) Check(ctx context.Context, baseline *Baseline) (Report, error) {
	if s == nil || baseline == nil {
		return Report{}, errors.New("drift sentinel and baseline are required")
	}
	if baseline.WorkspaceID != s.Workspace.Identity || baseline.WorkspaceRoot != s.Workspace.Root {
		return Report{}, errors.New("baseline belongs to another workspace")
	}
	current, err := collect(ctx, s.Workspace, baseline.Policy, s.Limits)
	if err != nil {
		return Report{}, err
	}
	report := Report{SchemaVersion: 1, BaselineID: baseline.ID, WorkspaceRoot: s.Workspace.Root, GeneratedAt: time.Now().UTC(), FailOn: append([]Severity(nil), baseline.Policy.FailOn...)}
	before := snapshotsByPath(baseline.Files)
	after := snapshotsByPath(current)
	for path, old := range before {
		newFile, ok := after[path]
		if !ok {
			report.Findings = append(report.Findings, Finding{Severity: SeverityError, Code: "file_deleted", Path: path, Message: "tracked source file was deleted", Before: old.SHA256})
			continue
		}
		if old.SHA256 == newFile.SHA256 {
			continue
		}
		report.ChangedFiles++
		report.Findings = append(report.Findings, Finding{Severity: SeverityWarning, Code: "file_changed", Path: path, Message: "source file content changed", Before: old.SHA256, After: newFile.SHA256})
		compareSnapshot(&report, old, newFile)
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			report.ChangedFiles++
			report.Findings = append(report.Findings, Finding{Severity: SeverityWarning, Code: "file_added", Path: path, Message: "new source file is outside the baseline", After: path})
		}
	}
	if baseline.Policy.MaxChangedFiles > 0 && report.ChangedFiles > baseline.Policy.MaxChangedFiles {
		report.Findings = append(report.Findings, Finding{Severity: SeverityError, Code: "changed_file_limit", Message: fmt.Sprintf("changed file count %d exceeds policy limit %d", report.ChangedFiles, baseline.Policy.MaxChangedFiles)})
	}
	checkArchitecture(&report, current, baseline.Policy)
	sort.Slice(report.Findings, func(i, j int) bool {
		left, right := report.Findings[i], report.Findings[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.Severity != right.Severity {
			return left.Severity < right.Severity
		}
		if left.Message != right.Message {
			return left.Message < right.Message
		}
		if fmt.Sprint(left.Before) != fmt.Sprint(right.Before) {
			return fmt.Sprint(left.Before) < fmt.Sprint(right.Before)
		}
		return fmt.Sprint(left.After) < fmt.Sprint(right.After)
	})
	report.Drifted = len(report.Findings) > 0
	return report, nil
}

func snapshotsByPath(files []FileSnapshot) map[string]FileSnapshot {
	out := make(map[string]FileSnapshot, len(files))
	for _, file := range files {
		out[file.Path] = file
	}
	return out
}

func compareSnapshot(report *Report, old, current FileSnapshot) {
	oldSymbols, newSymbols := symbolsByName(old.Symbols), symbolsByName(current.Symbols)
	for name, symbol := range oldSymbols {
		newSymbol, ok := newSymbols[name]
		if !ok {
			report.Findings = append(report.Findings, Finding{Severity: SeverityError, Code: "symbol_removed", Path: current.Path, Message: fmt.Sprintf("exported symbol %q was removed", name), Before: symbol.Signature})
			continue
		}
		if symbol.Signature != newSymbol.Signature {
			report.Findings = append(report.Findings, Finding{Severity: SeverityError, Code: "signature_changed", Path: current.Path, Message: fmt.Sprintf("exported symbol %q signature changed", name), Before: symbol.Signature, After: newSymbol.Signature})
		}
	}
	for name := range newSymbols {
		if _, ok := oldSymbols[name]; !ok {
			report.Findings = append(report.Findings, Finding{Severity: SeverityInfo, Code: "symbol_added", Path: current.Path, Message: fmt.Sprintf("exported symbol %q was added", name)})
		}
	}
	compareImports(report, old, current)
}

func symbolsByName(symbols []repomap.Symbol) map[string]repomap.Symbol {
	out := make(map[string]repomap.Symbol, len(symbols))
	for _, symbol := range symbols {
		out[symbol.Name] = symbol
	}
	return out
}

func compareImports(report *Report, old, current FileSnapshot) {
	oldSet, newSet := stringSet(old.Imports), stringSet(current.Imports)
	for imported := range oldSet {
		if _, ok := newSet[imported]; !ok {
			report.Findings = append(report.Findings, Finding{Severity: SeverityWarning, Code: "import_removed", Path: current.Path, Message: "dependency import removed", Before: imported})
		}
	}
	for imported := range newSet {
		if _, ok := oldSet[imported]; !ok {
			report.Findings = append(report.Findings, Finding{Severity: SeverityError, Code: "import_added", Path: current.Path, Message: "dependency import added", After: imported})
		}
	}
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[strings.TrimSpace(value)] = struct{}{}
	}
	return set
}
