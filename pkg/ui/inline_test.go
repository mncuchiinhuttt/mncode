package ui

import (
	"testing"
)

func TestGetMatchingSlashOptions(t *testing.T) {
	// 1. When query is "/" -> all options returned
	all := getMatchingSlashOptions(nil, "/")
	if len(all) == 0 {
		t.Errorf("expected options for '/', got 0")
	}

	// 2. When query is "/sk" -> "/skills" matched
	matched := getMatchingSlashOptions(nil, "/sk")
	if len(matched) == 0 || matched[0].Command != "/skills" {
		t.Errorf("expected '/skills' for query '/sk', got %v", matched)
	}

	// 3. When query is "/acc" -> "/account list" and "/account import" matched
	accMatched := getMatchingSlashOptions(nil, "/acc")
	if len(accMatched) < 2 {
		t.Errorf("expected at least 2 account commands for '/acc', got %d", len(accMatched))
	}

	// 4. Truncate test
	truncated := truncateText("A very long description text that needs truncation", 20)
	if len(truncated) > 20 {
		t.Errorf("expected max length 20, got %d (%s)", len(truncated), truncated)
	}

	// 5. Sliding window bounds test
	total := len(all)
	maxDisplay := 5
	for selectedIdx := 0; selectedIdx < total; selectedIdx++ {
		startIdx := 0
		if selectedIdx >= maxDisplay {
			startIdx = selectedIdx - maxDisplay + 1
		}
		endIdx := startIdx + maxDisplay
		if endIdx > total {
			endIdx = total
			startIdx = endIdx - maxDisplay
			if startIdx < 0 {
				startIdx = 0
			}
		}

		if selectedIdx < startIdx || selectedIdx >= endIdx {
			t.Errorf("selectedIdx %d is out of visible window [%d, %d)", selectedIdx, startIdx, endIdx)
		}
		if endIdx-startIdx > maxDisplay {
			t.Errorf("window size %d exceeds maxDisplay %d", endIdx-startIdx, maxDisplay)
		}
	}
}
