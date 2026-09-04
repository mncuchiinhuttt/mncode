package drift

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
)

// DefaultPolicy enables cycle detection while leaving layer rules opt-in.
func DefaultPolicy() Policy { return Policy{DenyCycles: true} }

// LoadPolicy reads an optional JSON policy from the workspace.
func LoadPolicy(workspace, explicit string) (Policy, string, error) {
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return Policy{}, "", err
	}
	path := explicit
	if strings.TrimSpace(path) == "" {
		path = filepath.Join(".mncode", "drift", "policy.json")
		if _, statErr := os.Stat(filepath.Join(root.Root, path)); os.IsNotExist(statErr) {
			return DefaultPolicy(), "", nil
		}
	}
	resolved, err := tools.ResolveWorkspacePath(root.Root, path, false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && explicit == "" {
			return DefaultPolicy(), "", nil
		}
		return Policy{}, "", err
	}
	var policy Policy
	if err := commandutil.ReadJSON(resolved, &policy, 512*1024); err != nil {
		return Policy{}, resolved, err
	}
	return policy, resolved, nil
}

// SavePolicy writes a policy only when the destination is workspace-bound.
func SavePolicy(workspace string, policy Policy) (string, error) {
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return "", err
	}
	path, err := tools.ResolveWorkspacePath(root.Root, filepath.Join(".mncode", "drift", "policy.json"), true)
	if err != nil {
		return "", err
	}
	if err := commandutil.WritePrivateJSON(path, policy); err != nil {
		return "", err
	}
	return path, nil
}

func checkArchitecture(report *Report, files []FileSnapshot, policy Policy) {
	if policy.MaxImportEdges > 0 {
		edges := 0
		for _, file := range files {
			edges += len(file.Imports)
		}
		if edges > policy.MaxImportEdges {
			report.Findings = append(report.Findings, Finding{Severity: SeverityError, Code: "import_limit", Path: "", Message: fmt.Sprintf("import graph has %d edges; policy allows %d", edges, policy.MaxImportEdges)})
		}
	}
	graph := buildGraph(files)
	for source, targets := range graph {
		layer := layerFor(source, policy.Layers)
		for target, raw := range targets {
			targetLayer := layerFor(target, policy.Layers)
			for _, forbidden := range policy.ForbiddenImports[layer] {
				if matchImport(forbidden, target, targetLayer, raw) {
					report.Findings = append(report.Findings, Finding{Severity: SeverityError, Code: "forbidden_import", Path: source, Message: fmt.Sprintf("%s layer imports forbidden %s (%s)", layer, target, raw)})
				}
			}
		}
	}
	if policy.DenyCycles {
		for _, cycle := range findCycles(graph) {
			report.Findings = append(report.Findings, Finding{Severity: SeverityError, Code: "import_cycle", Path: cycle[0], Message: "local import cycle: " + strings.Join(cycle, " -> ")})
		}
	}
}
