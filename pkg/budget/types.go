package budget

import (
	"fmt"
	"strconv"
	"strings"
)

// BudgetSpec defines a session token or cost ceiling.
type BudgetSpec struct {
	TokenLimit  int64   `json:"tokenLimit"`
	DollarLimit float64 `json:"dollarLimit,omitempty"`
	IsHardStop  bool    `json:"isHardStop"`
}

// ParseBudget parses user input like "100k", "200k!", "$5.00", "50000".
func ParseBudget(input string) (BudgetSpec, error) {
	input = strings.TrimSpace(input)
	if input == "" || strings.EqualFold(input, "clear") || strings.EqualFold(input, "none") {
		return BudgetSpec{TokenLimit: 0, IsHardStop: false}, nil
	}

	isHard := strings.HasSuffix(input, "!")
	trimmed := strings.TrimSuffix(input, "!")

	// Dollar syntax: $5.00
	if strings.HasPrefix(trimmed, "$") {
		dollarStr := strings.TrimPrefix(trimmed, "$")
		val, err := strconv.ParseFloat(dollarStr, 64)
		if err != nil || val <= 0 {
			return BudgetSpec{}, fmt.Errorf("invalid dollar budget %q: %w", input, err)
		}
		// Approximate token equivalent (e.g. $1 ~ 300k blended tokens)
		tokenEquiv := int64(val * 300000)
		return BudgetSpec{TokenLimit: tokenEquiv, DollarLimit: val, IsHardStop: isHard}, nil
	}

	// Suffix multipliers: 100k, 1m
	multiplier := int64(1)
	lower := strings.ToLower(trimmed)
	if strings.HasSuffix(lower, "k") {
		multiplier = 1000
		lower = strings.TrimSuffix(lower, "k")
	} else if strings.HasSuffix(lower, "m") {
		multiplier = 1000000
		lower = strings.TrimSuffix(lower, "m")
	}

	val, err := strconv.ParseInt(lower, 10, 64)
	if err != nil || val <= 0 {
		return BudgetSpec{}, fmt.Errorf("invalid token budget %q: %w", input, err)
	}

	return BudgetSpec{TokenLimit: val * multiplier, IsHardStop: isHard}, nil
}
