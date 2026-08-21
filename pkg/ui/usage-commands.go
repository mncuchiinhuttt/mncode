package ui

import (
	"fmt"
	"mncode/pkg/agent"
)

// ShowUsageStats opens the interactive Claude Code Usage Dashboard
func ShowUsageStats(s *agent.Session) {
	OpenInteractiveUsageDashboard(s)
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + string(make([]byte, width-len(s)))
}

func formatNumber(n int64) string {
	in := fmt.Sprintf("%d", n)
	out := ""
	for i, c := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	return out
}
