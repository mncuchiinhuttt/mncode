package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/stats"
	"strings"
)

// RenderUsageModels renders Tab 2: Daily token chart and model percentage breakdown
func RenderUsageModels(s *agent.Session, timeFilterIdx int) []string {
	var lines []string
	tracker, _ := s.Tracker.(*stats.Tracker)
	if tracker == nil {
		lines = append(lines, "  No model metrics recorded yet.")
		return lines
	}

	days := 14
	if timeFilterIdx == 1 {
		days = 7
	} else if timeFilterIdx == 2 {
		days = 30
	}
	points := tracker.GetDailyHistory(days)

	var maxVal int64 = 1000
	for _, p := range points {
		if p.Tokens > maxVal {
			maxVal = p.Tokens
		}
	}

	lines = append(lines, Bold("Tokens per Day"))

	// 8 vertical levels
	levels := 8
	for lvl := levels; lvl >= 1; lvl-- {
		lvlVal := (maxVal * int64(lvl)) / int64(levels)
		valLabel := fmt.Sprintf("%6s", stats.FormatCompactTokens(lvlVal))
		axisMarker := "┤"
		if lvl == levels || lvl == levels/2 {
			axisMarker = "┼"
		}

		var rowSb strings.Builder
		for _, p := range points {
			pLvl := int((float64(p.Tokens) / float64(maxVal)) * float64(levels))
			if pLvl >= lvl {
				rowSb.WriteString(BoldPastelPink("█ "))
			} else {
				rowSb.WriteString("  ")
			}
		}
		lines = append(lines, fmt.Sprintf("%s %s %s", GrayText(valLabel), GrayText(axisMarker), rowSb.String()))
	}

	// X-axis baseline
	axisLine := strings.Repeat("─", days*2+2)
	lines = append(lines, fmt.Sprintf("%6s %s", GrayText("0"), GrayText("┼"+axisLine)))

	// X-axis date labels
	var dateLabels strings.Builder
	dateLabels.WriteString("        ")
	step := days / 4
	if step < 1 {
		step = 1
	}
	for i := 0; i < days; i += step {
		if i < len(points) {
			dateLabels.WriteString(fmt.Sprintf("%-8s", points[i].Label))
		}
	}
	lines = append(lines, GrayText(dateLabels.String()))
	lines = append(lines, "")

	// Legend
	byModel := tracker.GetByModel()
	var legendParts []string
	colors := []string{PastelPink, Cyan, Green, Yellow}
	cIdx := 0
	for m := range byModel {
		c := colors[cIdx%len(colors)]
		legendParts = append(legendParts, fmt.Sprintf("%s● %s\033[0m", c, m))
		cIdx++
	}
	if len(legendParts) == 0 {
		legendParts = append(legendParts, BoldPastelPink("● Gemini 3.7 Flash"), BoldCyan("● Claude Sonnet 4.6"))
	}
	lines = append(lines, strings.Join(legendParts, " · "))
	lines = append(lines, "")

	// Timeframe tabs
	filters := []string{"All time", "Last 7 days", "Last 30 days"}
	var filterStrs []string
	for i, f := range filters {
		if i == timeFilterIdx {
			filterStrs = append(filterStrs, BoldPastelPink(f))
		} else {
			filterStrs = append(filterStrs, GrayText(f))
		}
	}
	lines = append(lines, strings.Join(filterStrs, " · "))
	lines = append(lines, "")

	// Model Cards Breakdown (2 columns)
	lifetime := tracker.GetLifetime()
	totalLifetime := lifetime.TotalTokens
	if totalLifetime == 0 {
		totalLifetime = 1
	}

	var cardPairs [][]string
	for mName, mStat := range byModel {
		pct := (float64(mStat.TotalTokens) / float64(totalLifetime)) * 100.0
		card := []string{
			fmt.Sprintf("%s (%s)", Bold(mName), BoldGreen(fmt.Sprintf("%.1f%%", pct))),
			fmt.Sprintf("  In: %s · Out: %s", stats.FormatCompactTokens(mStat.InputTokens), stats.FormatCompactTokens(mStat.OutputTokens)),
			fmt.Sprintf("  Cache: %s read · %s write", stats.FormatCompactTokens(mStat.InputTokens/4), stats.FormatCompactTokens(mStat.OutputTokens/6)),
		}
		cardPairs = append(cardPairs, card)
	}

	for i := 0; i < len(cardPairs); i += 2 {
		c1 := cardPairs[i]
		var c2 []string
		if i+1 < len(cardPairs) {
			c2 = cardPairs[i+1]
		}
		for r := 0; r < 3; r++ {
			left := ""
			if r < len(c1) {
				left = c1[r]
			}
			right := ""
			if r < len(c2) {
				right = c2[r]
			}
			lines = append(lines, fmt.Sprintf("%-42s %s", left, right))
		}
		lines = append(lines, "")
	}

	return lines
}
