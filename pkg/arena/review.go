package arena

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"mncode/pkg/commandutil"
)

var roles = []string{"security adversary", "correctness adversary", "maintainability adversary"}

// Arena coordinates independent hostile reviews and deterministic merging.
type Arena struct {
	Workspace commandutil.Workspace
	Reviewer  Reviewer
	Limits    commandutil.Limits
}

// New creates a red-team arena rooted at a canonical workspace.
func New(workspace string, reviewer Reviewer) (*Arena, error) {
	root, err := commandutil.ResolveWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	return &Arena{Workspace: root, Reviewer: reviewer, Limits: commandutil.DefaultLimits()}, nil
}

// Review runs the three adversarial roles concurrently and merges duplicates.
func (a *Arena) Review(ctx context.Context, source Source, opts Options) (Report, error) {
	if a == nil || a.Reviewer == nil {
		return Report{}, errors.New("arena reviewer is unavailable")
	}
	if strings.TrimSpace(source.Diff) == "" {
		return Report{}, errors.New("diff is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.MaxDiffBytes > 0 && int64(len(source.Diff)) > opts.MaxDiffBytes {
		return Report{}, fmt.Errorf("diff exceeds %d bytes", opts.MaxDiffBytes)
	}
	if source.DiffSHA256 == "" {
		source.DiffSHA256 = diffDigest(source.Diff)
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 90 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	started := time.Now().UTC()
	rounds := opts.Rounds
	if rounds <= 0 {
		rounds = 1
	}
	if rounds > 3 {
		rounds = 3
	}
	findings := make(chan []Finding, len(roles)*rounds)
	errs := make(chan error, len(roles)*rounds)
	var wg sync.WaitGroup
	for round := 1; round <= rounds; round++ {
		for _, role := range roles {
			wg.Add(1)
			go func(role string, round int) {
				defer wg.Done()
				found, err := a.Reviewer.Review(runCtx, source, role)
				if err != nil {
					errs <- fmt.Errorf("%s round %d: %w", role, round, err)
					return
				}
				findings <- found
			}(role, round)
		}
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-runCtx.Done():
		return Report{}, runCtx.Err()
	}
	close(findings)
	close(errs)
	for err := range errs {
		return Report{}, err
	}
	all := make([]Finding, 0)
	for batch := range findings {
		all = append(all, batch...)
	}
	merged := mergeFindings(all)
	report := Report{SchemaVersion: 1, ID: commandutil.NewID("arena"), Source: source, Findings: merged, StartedAt: started, EndedAt: time.Now().UTC(), Verdict: verdict(merged)}
	return report, nil
}

func mergeFindings(findings []Finding) []Finding {
	merged := make(map[string]Finding)
	for _, finding := range findings {
		finding.Severity = normalizeSeverity(finding.Severity)
		finding.File = strings.TrimSpace(finding.File)
		finding.Evidence = commandutil.Scrub(finding.Evidence)
		finding.Impact = commandutil.Scrub(finding.Impact)
		finding.Recommendation = commandutil.Scrub(finding.Recommendation)
		key := fmt.Sprintf("%s|%s|%d", strings.ToLower(finding.File), strings.ToLower(finding.Category), finding.Line)
		if key == "||0" {
			key = fmt.Sprintf("%s|%s", strings.ToLower(finding.Severity), finding.Evidence)
		}
		if old, ok := merged[key]; !ok || severityRank(finding.Severity) > severityRank(old.Severity) || severityRank(finding.Severity) == severityRank(old.Severity) && findingLess(finding, old) {
			finding.ID = stableFindingID(key)
			merged[key] = finding
		}
	}
	out := make([]Finding, 0, len(merged))
	for _, finding := range merged {
		out = append(out, finding)
	}
	sort.Slice(out, func(i, j int) bool {
		if severityRank(out[i].Severity) != severityRank(out[j].Severity) {
			return severityRank(out[i].Severity) > severityRank(out[j].Severity)
		}
		return findingLess(out[i], out[j])
	})
	return out
}
func findingLess(left, right Finding) bool {
	leftKey := strings.Join([]string{left.Category, left.File, fmt.Sprint(left.Line), left.Evidence, left.Recommendation}, "\x00")
	rightKey := strings.Join([]string{right.Category, right.File, fmt.Sprint(right.Line), right.Evidence, right.Recommendation}, "\x00")
	return leftKey < rightKey
}

func normalizeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical", "high", "block":
		return "high"
	case "medium", "warn", "warning":
		return "medium"
	case "low", "info", "note":
		return "low"
	default:
		return "medium"
	}
}
func severityRank(value string) int {
	switch value {
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}
func verdict(findings []Finding) string {
	for _, finding := range findings {
		if finding.Severity == "high" {
			return "block"
		}
	}
	if len(findings) > 0 {
		return "warn"
	}
	return "pass"
}

// ParseFindings parses the stable pipe-delimited reviewer protocol.
func ParseFindings(text, role string) []Finding {
	var findings []Finding
	for _, line := range strings.Split(text, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 8)
		if len(parts) != 8 || strings.ToUpper(parts[0]) != "FINDING" {
			continue
		}
		lineNumber := 0
		_, _ = fmt.Sscanf(strings.TrimSpace(parts[3]), "%d", &lineNumber)
		findings = append(findings, Finding{ID: commandutil.NewID("finding"), Severity: parts[1], File: parts[2], Line: lineNumber, Category: parts[4], Evidence: parts[5], Impact: parts[6], Recommendation: parts[7], Confidence: 0.7})
	}
	if len(findings) == 0 && strings.TrimSpace(text) != "" {
		findings = append(findings, Finding{ID: commandutil.NewID("review"), Severity: "low", Category: "unstructured-review", Evidence: fmt.Sprintf("%s reviewer returned no structured findings", role), Impact: "Review output cannot be merged into the risk matrix.", Recommendation: "Re-run the arena with a reviewer that follows the FINDING protocol.", Confidence: 0.2})
	}
	return findings
}

func diffDigest(diff string) string {
	digest := sha256.Sum256([]byte(diff))
	return hex.EncodeToString(digest[:])
}
func stableFindingID(key string) string {
	digest := sha256.Sum256([]byte(key))
	return "finding-" + hex.EncodeToString(digest[:8])
}
