package budget

import (
	"strings"
	"testing"
)

func TestParseBudget(t *testing.T) {
	cases := []struct {
		input       string
		wantTokens  int64
		wantHard    bool
		wantDollar  float64
		wantErr     bool
	}{
		{"100k", 100000, false, 0, false},
		{"250k!", 250000, true, 0, false},
		{"1m", 1000000, false, 0, false},
		{"50000", 50000, false, 0, false},
		{"$2.50", 750000, false, 2.50, false},
		{"$5.00!", 1500000, true, 5.00, false},
		{"clear", 0, false, 0, false},
		{"none", 0, false, 0, false},
		{"invalid-budget", 0, false, 0, true},
	}

	for _, c := range cases {
		spec, err := ParseBudget(c.input)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseBudget(%q) expected error, got nil", c.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseBudget(%q) unexpected error: %v", c.input, err)
			continue
		}
		if spec.TokenLimit != c.wantTokens {
			t.Errorf("ParseBudget(%q) TokenLimit = %d, want %d", c.input, spec.TokenLimit, c.wantTokens)
		}
		if spec.IsHardStop != c.wantHard {
			t.Errorf("ParseBudget(%q) IsHardStop = %v, want %v", c.input, spec.IsHardStop, c.wantHard)
		}
	}
}

func TestTrackerEnforcement(t *testing.T) {
	spec, _ := ParseBudget("100k!")
	tracker := NewTracker(spec)

	// Step 1: Add 50k tokens -> no warning, not hard
	hard, notice := tracker.AddTokens(25000, 20000, 5000)
	if hard || notice != "" {
		t.Fatalf("unexpected alert at 50%%: hard=%v, notice=%q", hard, notice)
	}

	// Step 2: Add 35k tokens -> reach 85k (85%) -> triggers 80% advisory
	hard, notice = tracker.AddTokens(15000, 15000, 5000)
	if hard || !strings.Contains(notice, "80%") {
		t.Fatalf("expected 80%% warning, got hard=%v, notice=%q", hard, notice)
	}

	// Step 3: Add 20k tokens -> reach 105k (105%) -> triggers hard stop
	hard, notice = tracker.AddTokens(10000, 8000, 2000)
	if !hard || !strings.Contains(notice, "HARD STOP") {
		t.Fatalf("expected hard stop alert, got hard=%v, notice=%q", hard, notice)
	}

	if !tracker.IsHardStopExceeded() {
		t.Fatal("expected IsHardStopExceeded() to be true")
	}

	rem, unlimited := tracker.Remaining()
	if unlimited || rem != 0 {
		t.Fatalf("expected remaining=0, unlimited=false, got %d, %v", rem, unlimited)
	}
}
